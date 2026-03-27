package stocks

type Sector uint8

const (
	BensIndustriais Sector = iota + 1
	Comunicacoes
	ConstrucaoSuporte
	ConsumoCiclico
	ConsumoNCiclico
	Financeiro
	MateriaisBasicos
	Outros
	PetroleoGasBioComb
	Saude
	SetorInicial
	TI
	UtilPublica
)

func (sect Sector) String() string {
	switch sect {
	case BensIndustriais:
		return "Bens Industriais"
	case Comunicacoes:
		return "Comunicações"
	case ConstrucaoSuporte:
		return "Construção e Suporte"
	case ConsumoCiclico:
		return "Consumo Cíclico"
	case ConsumoNCiclico:
		return "Consumo Não Cíclico"
	case Financeiro:
		return "Financeiro"
	case MateriaisBasicos:
		return "Materiais Básicos"
	case Outros:
		return "Outros"
	case PetroleoGasBioComb:
		return "Petróleo, Gás e Biocombustíveis"
	case Saude:
		return "Saúde"
	case SetorInicial:
		return "Setor Inicial"
	case TI:
		return "Tecnologia da Informação"
	case UtilPublica:
		return "Utilidade Pública"
	default:
		return ""
	}
}
