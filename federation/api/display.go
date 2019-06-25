package api

// Display defines how the data is displayed
type Display struct {
	Filter map[string]interface{} `json:"filter"`
	Sorter Sorter                 `json:"sort"`
}

type Sorter struct {
	By    string `json:"by"`
	Order string `json:"order"`
}
