package handler

import (
	"belajar-go/challenge/transactionSystem/internal/mocks"
	// "belajar-go/challenge/transactionSystem/internal/models"
	// "bytes"
	// "encoding/json"
	// "io"
	// "net/http"
	// "net/http/httptest"
	// "testing"
	// "time"
	// "github.com/google/uuid"
	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock"
	// "github.com/stretchr/testify/mock"
)

type Mocker struct {
	mockService *mocks.BankService
}

var idTest string = "d369cc5a-c96e-4386-b01e-086bcbc44d00"

// func TestBanksHandler_GetAll(t *testing.T) {
// 	testCases := []struct {
// 		desc      string
// 		mockSetup func(m *Mocker)
// 		expected  []models.Bank
// 		wantErr   bool
// 	}{
// 		{
// 			desc: "SUCCESS: Get All Banks",
// 			mockSetup: func(m *Mocker) {
// 				fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
// 				m.mockService.On("FetchAllBanks").Return([]models.Bank{
// 					{ID: uuid.MustParse("d369cc5a-c96e-4386-b01e-086bcbc44d00"), BankCode: "CIMB", BankName: "CIMB NIAGA", CreatedAt: fixedTime},
// 				}, nil)
// 			},
// 			expected: []models.Bank{
// 				{ID: uuid.MustParse("d369cc5a-c96e-4386-b01e-086bcbc44d00"), BankCode: "CIMB", BankName: "CIMB NIAGA", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			desc: "ERROR: Error When get all banks",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("FetchAllBanks").Return(nil, assert.AnError)
// 			},
// 			expected: nil,
// 			wantErr:  true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			m := &Mocker{
// 				mockService: mocks.NewBankService(t),
// 			}
// 			tc.mockSetup(m)

// 			h := &BanksHandler{svc: m.mockService}

// 			req := httptest.NewRequest(http.MethodGet, "/banks", nil)
// 			w := httptest.NewRecorder()

// 			h.GetAll()(w, req)

// 			if tc.wantErr {
// 				assert.NotEqual(t, http.StatusOK, w.Code)
// 			} else {
// 				assert.Equal(t, http.StatusOK, w.Code)

// 				var response struct {
// 					Message string `json:"message"`
// 					Data    struct {
// 						Bank []models.Bank `json:"banks"`
// 					} `json:"data"`
// 				}

// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				assert.NoError(t, err)

// 				assert.EqualValues(t, tc.expected, response.Data.Bank)
// 			}

// 			m.mockService.AssertExpectations(t)
// 		})
// 	}
// }

// func TestBanksHandler_Create(t *testing.T) {
// 	testCases := []struct {
// 		desc      string
// 		mockSetup func(m *Mocker)
// 		expected  *models.Bank
// 		wantErr   bool
// 	}{
// 		{
// 			desc: "SUCCESS: Create new bank",
// 			mockSetup: func(m *Mocker) {
// 				fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// 				m.mockService.On("CreateNewBank", mock.Anything).Return(models.Bank{
// 					ID: uuid.MustParse("d369cc5a-c96e-4386-b01e-086bcbc44d00"), BankCode: "CIMB", BankName: "CIMB NIAGA", CreatedAt: fixedTime,
// 				}, nil)
// 			},
// 			expected: &models.Bank{
// 				ID: uuid.MustParse("d369cc5a-c96e-4386-b01e-086bcbc44d00"), BankCode: "CIMB", BankName: "CIMB NIAGA", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			desc: "ERROR: Error When creating new bank",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("CreateNewBank", mock.Anything).Return(models.Bank{}, assert.AnError)
// 			},
// 			expected: nil,
// 			wantErr:  true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			m := &Mocker{
// 				mockService: mocks.NewBankService(t),
// 			}
// 			tc.mockSetup(m)

// 			h := &BanksHandler{svc: m.mockService}

// 			var body io.Reader
// 			if !tc.wantErr && tc.expected != nil {
// 				jsonByte, _ := json.Marshal(tc.expected)
// 				body = bytes.NewBuffer(jsonByte)
// 			} else {
// 				body = bytes.NewBuffer([]byte("{}"))
// 			}

// 			req := httptest.NewRequest(http.MethodPost, "/bank", body)
// 			w := httptest.NewRecorder()

// 			h.Create()(w, req)

// 			if tc.wantErr {
// 				assert.NotEqual(t, http.StatusCreated, w.Code)
// 			} else {
// 				assert.Equal(t, http.StatusCreated, w.Code)

// 				var response struct {
// 					Message string `json:"message"`
// 					Data    struct {
// 						Bank models.Bank `json:"bank"`
// 					} `json:"data"`
// 				}

// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				assert.NoError(t, err)

// 				assert.EqualValues(t, *tc.expected, response.Data.Bank)
// 			}

// 			m.mockService.AssertExpectations(t)
// 		})
// 	}
// }

// func TestBanksHandler_Patch(t *testing.T) {
// 	testCases := []struct {
// 		desc      string
// 		mockSetup func(m *Mocker)
// 		expected  string
// 		wantErr   bool
// 	}{
// 		{
// 			desc: "SUCCESS: Update bank by id",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("PatchBank", mock.Anything).Return("d369cc5a-c96e-4386-b01e-086bcbc44d00", nil)
// 			},
// 			expected: "d369cc5a-c96e-4386-b01e-086bcbc44d00",
// 			wantErr:  false,
// 		},
// 		{
// 			desc: "ERROR: Error When updating bank by id",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("PatchBank", mock.Anything).Return("", assert.AnError)
// 			},
// 			expected: "",
// 			wantErr:  true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			m := &Mocker{
// 				mockService: mocks.NewBankService(t),
// 			}
// 			tc.mockSetup(m)

// 			h := &BanksHandler{svc: m.mockService}

// 			reqBody := models.Bank{BankCode: "CIMB", BankName: "CIMB NIAGA"}
// 			jsonByte, _ := json.Marshal(reqBody)
// 			body := bytes.NewBuffer(jsonByte)

// 			req := httptest.NewRequest(http.MethodPatch, "/bank/d369cc5a-c96e-4386-b01e-086bcbc44d00", body)
// 			w := httptest.NewRecorder()

// 			mux := http.NewServeMux()
// 			mux.HandleFunc("PATCH /bank/{id}", h.Update())
// 			mux.ServeHTTP(w, req)

// 			if tc.wantErr {
// 				assert.NotEqual(t, http.StatusPartialContent, w.Code)
// 			} else {
// 				assert.Equal(t, http.StatusPartialContent, w.Code)

// 				var response struct {
// 					Message string         `json:"message"`
// 					Data    map[string]any `json:"data"`
// 				}

// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				assert.NoError(t, err)
// 				assert.EqualValues(t, tc.expected, response.Data["id"])
// 			}

// 			m.mockService.AssertExpectations(t)
// 		})
// 	}
// }

// func TestBanksHandler_Delete(t *testing.T) {
// 	testCases := []struct {
// 		desc         string
// 		mockSetup    func(m *Mocker)
// 		expectedCode int
// 		wantErr      bool
// 	}{
// 		{
// 			desc: "SUCCESS: Delete bank by id",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("DeleteBank", idTest).Return(nil)
// 			},
// 			expectedCode: http.StatusOK,
// 			wantErr:      false,
// 		},
// 		{
// 			desc: "ERROR: Error When delete bank by id",
// 			mockSetup: func(m *Mocker) {
// 				m.mockService.On("DeleteBank", idTest).Return("", assert.AnError)
// 			},
// 			expectedCode: http.StatusBadRequest,
// 			wantErr:      true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			m := &Mocker{
// 				mockService: mocks.NewBankService(t),
// 			}
// 			tc.mockSetup(m)

// 			h := &BanksHandler{svc: m.mockService}

// 			req := httptest.NewRequest(http.MethodDelete, "/bank/"+idTest, nil)
// 			w := httptest.NewRecorder()

// 			mux := http.NewServeMux()
// 			mux.HandleFunc("DELETE /bank/{id}", h.Delete())
// 			mux.ServeHTTP(w, req)

// 			if tc.wantErr {
// 				assert.NotEqual(t, tc.expectedCode, w.Code)
// 			} else {
// 				assert.Equal(t, tc.expectedCode, w.Code)

// 				var response struct {
// 					Message string         `json:"message"`
// 					Data    map[string]any `json:"data"`
// 				}

// 				err := json.Unmarshal(w.Body.Bytes(), &response)
// 				assert.NoError(t, err)
// 				assert.Equal(t, fmt.Sprintf("Berhasil menghapus bank : %s", idTest), response.Message)
// 			}

// 			m.mockService.AssertExpectations(t)
// 		})
// 	}
// }
