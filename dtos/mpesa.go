package dtos

type MpesaResultResponse struct {
	Result MpesaResult `json:"Result"`
}

type MpesaResult struct {
	ConversationID           string           `json:"ConversationID"`
	OriginatorConversationID string           `json:"OriginatorConversationID"`
	ReferenceData            ReferenceData    `json:"ReferenceData"`
	ResultCode               int              `json:"ResultCode"`
	ResultDesc               string           `json:"ResultDesc"`
	ResultParameters         ResultParameters `json:"ResultParameters"`
	ResultType               int              `json:"ResultType"`
	TransactionID            string           `json:"TransactionID"`
}

type ReferenceData struct {
	ReferenceItem ReferenceItem `json:"ReferenceItem"`
}

type ReferenceItem struct {
	Key string `json:"Key"`
}

type ResultParameters struct {
	ResultParameter []ResultParameter `json:"ResultParameter"`
}

type ResultParameter struct {
	Key   string `json:"Key"`
	Value string `json:"Value,omitempty"`
}

type TransactionStatusResponse struct {
	Status                   string `json:"status"`
	TransactionID            string `json:"transaction_id,omitempty"`
	Message                  string `json:"message,omitempty"`
	MpesaReference           string `json:"mpesa_reference"`
	ConversationID           string `json:"conversation_id"`
	OriginatorConversationID string `json:"originator_conversation_id"`
}

type MpesaTransactionStatusRequest struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}
