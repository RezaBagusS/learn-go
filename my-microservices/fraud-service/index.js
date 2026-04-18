require('dotenv').config();
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');
const Redis = require('ioredis');

const PROTO_PATH = path.join(__dirname, '../shared/pb/fraud/fraud.proto');

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const fraudProto = grpc.loadPackageDefinition(packageDefinition).fraud;

const redis = new Redis({
  host: process.env.REDIS_HOST || 'localhost',
  port: process.env.REDIS_PORT || 6379,
});

// Mock Data / Parametrized rules
const BLACKLISTED_ACCOUNTS = ['9999999999', '1234567890'];
const MAX_AMOUNT_LIMIT = 50000000; // 50 Juta

/**
 * Handle gRPC request ValidateTransaction
 */
async function validateTransaction(call, callback) {
  const req = call.request;
  console.log(`\n    gRPC Validation Request: ${req.transaction_id}`);
  console.log(`   From: ${req.sender_id} -> To: ${req.receiver_id} | Amount: ${req.amount}`);

  const startTime = Date.now();
  let isFraud = false;
  let reason = 'Safe';
  let action = 'ALLOW';

  try {
    // --- 1. Check Blacklist ---
    const isSharedBlacklist = await redis.sismember('fraud:blacklist', req.receiver_id);
    if (BLACKLISTED_ACCOUNTS.includes(req.receiver_id) || isSharedBlacklist) {
      isFraud = true;
      reason = 'Receiver is in fraud blacklist';
      action = 'BLOCK';
    }

    // --- 2. Check Daily Cumulative Limit (50jt/Day) ---
    if (!isFraud) {
      const today = new Date().toISOString().split('T')[0];
      const dailyLimitKey = `fraud:daily_limit:${req.sender_id}:${today}`;

      const amount = parseInt(req.amount);
      const currentDailyTotal = await redis.incrby(dailyLimitKey, amount);

      // Set expiry 24 jam jika key baru
      if (currentDailyTotal === amount) {
        await redis.expire(dailyLimitKey, 86400);
      }

      if (currentDailyTotal > MAX_AMOUNT_LIMIT) {
        isFraud = true;
        reason = `Daily limit exceeded: ${currentDailyTotal} / ${MAX_AMOUNT_LIMIT}`;
        action = 'BLOCK';
        // Rollback nominal karena transaksi ini gagal/dihambat
        await redis.incrby(dailyLimitKey, -amount);
      }
    }

    // --- 3. Velocity Check (Max 5 tx / minute) ---
    if (!isFraud) {
      const velocityKey = `fraud:velocity:${req.sender_id}`;
      const count = await redis.incr(velocityKey);
      if (count === 1) await redis.expire(velocityKey, 300);

      if (count > 5) {
        isFraud = true;
        reason = 'Velocity limit exceeded (max 5 tx/min)';
        action = 'BLOCK';
      }
    }

    // --- 4. Logic Check Jam (01:00 - 04:00) -> Review Only ---
    const hour = new Date().getHours();
    if (hour >= 1 && hour <= 4 && !isFraud) {
      reason = 'Suspicious hour (1AM - 4AM)';
      action = 'REVIEW';
    }

  } catch (err) {
    console.error('Validation Error:', err);
  }

  const duration = Date.now() - startTime;
  console.log(`   Result: ${action} (${reason}) | Time: ${duration}ms`);

  callback(null, {
    is_fraud: isFraud,
    reason: reason,
    action: action,
  });
}

/**
 * Start Server
 */
function main() {
  const server = new grpc.Server();
  server.addService(fraudProto.FraudService.service, { validateTransaction });

  const port = process.env.APP_PORT || '50051';
  server.bindAsync(`0.0.0.0:${port}`, grpc.ServerCredentials.createInsecure(), (err, port) => {
    if (err) {
      console.error('Failed to bind gRPC server:', err);
      return;
    }
    console.log(`🚀 Fraud gRPC Service running at 0.0.0.0:${port}`);
    server.start();
  });
}

main();
