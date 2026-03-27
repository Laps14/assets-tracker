package stocks

type Stock struct {
	Title string
	Description string
	Sect Sector
	TotalSupply float32
	StockVal float32
	ID uint64
}
