package stocks

type Stock struct {
	Title string
	Description string
	Sect Sector
	TotalSupply float64
	StockVal float64
	ID uint64 // Is it necessary?
}
