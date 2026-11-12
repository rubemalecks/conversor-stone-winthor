package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Codigo da bandeira (campo 27) conforme a Tabela II do layout de retorno 4.1 da
// Software Express/Fiserv. A chave e o nome da bandeira no CSV da Stone, normalizado
// (minusculas, sem acentos e sem espacos).
var bandeiras = map[string]string{
	"americanexpress": "0001", "amex": "0001",
	"cabal":  "0002",
	"diners": "0003", "dinersclub": "0003",
	"elo":       "0004",
	"hipercard": "0005", "hiper": "0005",
	"mastercard": "0007", "master": "0007",
	"visa": "0008",
	"jcb":  "0030",
}

// Codigo da rede Stone (campo 18) na Tabela I do layout 4.1.
const redeStone = "369"

// normaliza deixa o texto em minusculas, sem acentos e sem espacos.
var semAcento = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e",
	"í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c", " ", "", "-", "", "_", "")

func normaliza(s string) string {
	return semAcento.Replace(strings.ToLower(strings.TrimSpace(s)))
}

// tipoProduto devolve o campo 16 do layout: C (credito) ou D (debito).
// Pre-pago segue o tipo base ("Debito Pre-pago" -> D).
func tipoProduto(produto string) (string, bool) {
	p := normaliza(produto)
	switch {
	case strings.HasPrefix(p, "debito"):
		return "D", true
	case strings.HasPrefix(p, "credito"):
		return "C", true
	}
	return "", false
}

const motivoCancelOutro = "ATENÇÃO: cancelamento de venda paga em outro período - estornar a baixa no WinThor"

var colunasStone = []string{"STONECODE", "CATEGORIA", "DATA DA VENDA", "DATA DE VENCIMENTO",
	"BANDEIRA", "PRODUTO", "STONE ID", "QTD DE PARCELAS", "Nº DA PARCELA",
	"VALOR BRUTO", "VALOR LÍQUIDO", "ÚLTIMO STATUS"}

// ArquivoDia e um arquivo gerado para uma data de recebimento.
type ArquivoDia struct {
	Data     string // AAAAMMDD
	Nome     string // nome sugerido: stone-1503-recebimento-DD-MM-AAAA.csv
	Conteudo []byte
	Caminho  string // onde foi salvo ("" se ainda nao foi salvo)
	Parcelas int
	Bruto    int64 // centavos
	Liquido  int64
}

// Gravar salva o arquivo no caminho escolhido.
func (a *ArquivoDia) Gravar(caminho string) error {
	if err := os.WriteFile(caminho, a.Conteudo, 0o644); err != nil {
		return fmt.Errorf("não consegui gravar %s: %v", filepath.Base(caminho), err)
	}
	a.Caminho = caminho
	return nil
}

// Resultado resume uma conversao.
type Resultado struct {
	Origem       string // CSV da Stone
	Pasta        string // pasta onde os arquivos foram salvos
	Arquivos     []ArquivoDia
	Conferencia  string // caminho do arquivo de conferencia ("" se nao houve)
	confConteudo []byte
	NaoConvert   int
	Motivos      map[string]int
	LinhasStone  int

	// CancelOutroPeriodo conta cancelamentos cuja venda nao esta no CSV
	// (venda ja paga e baixada antes): exigem estorno manual no WinThor.
	CancelOutroPeriodo int
}

// GravarConferencia salva, na pasta indicada, o CSV com as linhas que ficaram de fora.
func (r *Resultado) GravarConferencia(pasta string) error {
	if len(r.confConteudo) == 0 {
		return nil
	}
	base := strings.TrimSuffix(filepath.Base(r.Origem), filepath.Ext(r.Origem))
	caminho := filepath.Join(pasta, base+"-conferencia.csv")
	if err := os.WriteFile(caminho, r.confConteudo, 0o644); err != nil {
		return fmt.Errorf("não consegui gravar o arquivo de conferência: %v", err)
	}
	r.Conferencia = caminho
	return nil
}

// GravarTodos salva todos os arquivos (e a conferencia) numa pasta, com os nomes sugeridos.
func (r *Resultado) GravarTodos(pasta string) error {
	for i := range r.Arquivos {
		if err := r.Arquivos[i].Gravar(filepath.Join(pasta, r.Arquivos[i].Nome)); err != nil {
			return err
		}
	}
	r.Pasta = pasta
	return r.GravarConferencia(pasta)
}

// Totais soma os arquivos ja salvos.
func (r *Resultado) Totais() (parcelas int, bruto, liquido int64) {
	for _, a := range r.Arquivos {
		if a.Caminho == "" {
			continue
		}
		parcelas += a.Parcelas
		bruto += a.Bruto
		liquido += a.Liquido
	}
	return
}

// centavos converte "1.524,330000" em 152433, arredondando para 2 casas.
func centavos(txt string) (int64, error) {
	t := strings.TrimSpace(txt)
	neg := strings.HasPrefix(t, "-")
	t = strings.TrimPrefix(t, "-")
	t = strings.ReplaceAll(t, ".", "")
	inteiro, frac := t, ""
	if i := strings.Index(t, ","); i >= 0 {
		inteiro, frac = t[:i], t[i+1:]
	}
	if inteiro == "" {
		inteiro = "0"
	}
	for len(frac) < 3 {
		frac += "0"
	}
	n, err := strconv.ParseInt(inteiro+frac[:2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor inválido: %q", txt)
	}
	if frac[2] >= '5' {
		n++
	}
	if neg {
		n = -n
	}
	return n, nil
}

// reais formata centavos como 1.234,56.
func reais(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	s := strconv.FormatInt(c/100, 10)
	var b []byte
	for i, ch := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, '.')
		}
		b = append(b, ch)
	}
	out := fmt.Sprintf("%s,%02d", b, c%100)
	if neg {
		out = "-" + out
	}
	return out
}

// data8 converte "30/10/2026 14:53:22" em "20261030".
func data8(txt string) string {
	if len(txt) < 10 {
		return ""
	}
	return txt[6:10] + txt[3:5] + txt[0:2]
}

func dataBR(d8 string) string {
	if len(d8) != 8 {
		return d8
	}
	return d8[6:8] + "/" + d8[4:6] + "/" + d8[0:4]
}

func zfill(s string, n int) string {
	s = strings.TrimSpace(s)
	for len(s) < n {
		s = "0" + s
	}
	return s
}

func lerStone(caminho string) ([]map[string]string, error) {
	raw, err := os.ReadFile(caminho)
	if err != nil {
		return nil, err
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(raw) { // latin-1 -> utf-8
		r := make([]rune, len(raw))
		for i, b := range raw {
			r[i] = rune(b)
		}
		raw = []byte(string(r))
	}
	rd := csv.NewReader(bytes.NewReader(raw))
	rd.Comma = ';'
	rd.LazyQuotes = true
	rd.FieldsPerRecord = -1
	recs, err := rd.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("não consegui ler o CSV: %v", err)
	}
	if len(recs) < 2 {
		return nil, errors.New("o arquivo está vazio")
	}
	cab := map[string]int{}
	for i, c := range recs[0] {
		cab[strings.ToUpper(strings.TrimSpace(c))] = i
	}
	var faltando []string
	for _, c := range colunasStone {
		if _, ok := cab[c]; !ok {
			faltando = append(faltando, c)
		}
	}
	if len(faltando) > 0 {
		return nil, errors.New("este não parece ser o CSV de recebimentos da Stone.\nColunas não encontradas: " + strings.Join(faltando, ", "))
	}
	var linhas []map[string]string
	for _, r := range recs[1:] {
		if len(r) == 1 && strings.TrimSpace(r[0]) == "" {
			continue
		}
		m := map[string]string{}
		for _, c := range colunasStone {
			if i := cab[c]; i < len(r) {
				m[c] = strings.TrimSpace(r[i])
			}
		}
		linhas = append(linhas, m)
	}
	return linhas, nil
}

func aspas(campos []string) string {
	return `"` + strings.Join(campos, `";"`) + `"`
}

// Converter le o CSV da Stone e monta, em memoria, um arquivo no padrao 4.1 por
// data de recebimento, mais a conferencia com o que ficou de fora. Nada e gravado
// aqui: use ArquivoDia.Gravar / GravarConferencia / GravarTodos.
func Converter(caminho string) (*Resultado, error) {
	linhas, err := lerStone(caminho)
	if err != nil {
		return nil, err
	}
	res := &Resultado{Origem: caminho, Motivos: map[string]int{}, LinhasStone: len(linhas)}
	porData := map[string][][]string{}
	var conf [][]string

	// Vendas canceladas integralmente: mesmo Stone ID, parcela e valor de um cancelamento.
	// Guarda tambem as vendas do CSV, para achar cancelamentos de vendas de outro periodo.
	chave := func(s map[string]string, v int64) string {
		return fmt.Sprintf("%s|%s|%d", s["STONE ID"], s["Nº DA PARCELA"], v)
	}
	cancelada := map[string]bool{}
	vendaNoCSV := map[string]bool{}
	for _, s := range linhas {
		if s["STONE ID"] == "" {
			continue
		}
		v, err := centavos(s["VALOR BRUTO"])
		if err != nil {
			continue
		}
		switch s["CATEGORIA"] {
		case "Cancelamento":
			cancelada[chave(s, -v)] = true
		case "Venda":
			vendaNoCSV[chave(s, v)] = true
		}
	}

	for n, s := range linhas {
		motivo := ""
		band, okBand := bandeiras[normaliza(s["BANDEIRA"])]
		prod, okProd := tipoProduto(s["PRODUTO"])
		switch {
		case s["CATEGORIA"] == "Cancelamento":
			motivo = "Cancelamento (venda cancelada no mesmo período)"
			if v, err := centavos(s["VALOR BRUTO"]); err == nil && !vendaNoCSV[chave(s, -v)] {
				// A venda nao esta neste CSV: provavelmente ja foi paga e baixada antes.
				motivo = motivoCancelOutro
				res.CancelOutroPeriodo++
			}
		case s["CATEGORIA"] == "Balanceamento de Saldo":
			motivo = "Ajuste interno da Stone (Balanceamento de Saldo)"
		case s["CATEGORIA"] != "Venda":
			motivo = "Não é venda (" + s["CATEGORIA"] + ")"
		case s["ÚLTIMO STATUS"] != "Pago":
			motivo = "Ainda não pago pela Stone (" + s["ÚLTIMO STATUS"] + ")"
		case s["STONE ID"] == "":
			motivo = "Linha sem Stone ID"
		case !okBand:
			motivo = "Bandeira sem código conhecido (" + s["BANDEIRA"] + ")"
		case !okProd:
			motivo = "Produto sem tipo conhecido (" + s["PRODUTO"] + ")"
		case len(s["DATA DA VENDA"]) < 19 || len(s["DATA DE VENCIMENTO"]) < 10:
			motivo = "Data da venda ou de vencimento inválida"
		}
		var bruto, liquido int64
		if motivo == "" {
			var e1, e2 error
			bruto, e1 = centavos(s["VALOR BRUTO"])
			liquido, e2 = centavos(s["VALOR LÍQUIDO"])
			if e1 != nil || e2 != nil {
				return nil, fmt.Errorf("linha %d: valor inválido", n+2)
			}
			if bruto <= 0 {
				motivo = "Valor bruto zerado ou negativo"
			} else if cancelada[chave(s, bruto)] {
				motivo = "Venda cancelada (há cancelamento do mesmo valor)"
			}
		}
		if motivo != "" {
			res.Motivos[motivo]++
			conf = append(conf, []string{motivo, s["CATEGORIA"], s["DATA DA VENDA"], s["DATA DE VENCIMENTO"],
				s["BANDEIRA"], s["PRODUTO"], s["STONE ID"], s["QTD DE PARCELAS"], s["Nº DA PARCELA"],
				s["VALOR BRUTO"], s["VALOR LÍQUIDO"], s["ÚLTIMO STATUS"]})
			continue
		}
		taxa := bruto - liquido
		perc := (taxa*10000*2 + bruto) / (2 * bruto) // arredonda ao mais proximo
		qtd := s["QTD DE PARCELAS"]
		aVista := qtd == "" || qtd == "0" || qtd == "1"
		qtdOut, parcOut := "00", "00"
		if !aVista {
			qtdOut, parcOut = zfill(qtd, 2), zfill(s["Nº DA PARCELA"], 2)
		}
		dtVenda := data8(s["DATA DA VENDA"])
		hrVenda := strings.ReplaceAll(s["DATA DA VENDA"][11:19], ":", "")
		dtCred := data8(s["DATA DE VENCIMENTO"])
		sid := s["STONE ID"]
		// O NSU (campo 7) leva o proprio Stone ID: a 1503 localiza o titulo
		// comparando esse campo com PCPREST.NSUHOST.
		porData[dtCred] = append(porData[dtCred], []string{
			"10", "0", zfill(s["STONECODE"], 15), dtVenda, sid, sid, sid, "",
			strconv.FormatInt(bruto, 10), qtdOut, strconv.FormatInt(liquido, 10), "", dtCred, "", parcOut,
			prod, "0", redeStone, "0", "000000", "", strconv.FormatInt(taxa, 10), strconv.FormatInt(perc, 10),
			"", "", "", band, dtVenda, hrVenda, "", "", "0", hrVenda, "3", "0", dtVenda, sid, "0"})
	}

	datas := make([]string, 0, len(porData))
	for d := range porData {
		datas = append(datas, d)
	}
	sort.Strings(datas)
	agora := time.Now()
	for _, d := range datas {
		regs := porData[d]
		var b strings.Builder
		b.WriteString(aspas([]string{"0", agora.Format("20060102"), agora.Format("150405"), d, d,
			"V4.1", "0", agora.Format("20060102150405"), "000001"}) + "\n")
		a := ArquivoDia{Data: d, Parcelas: len(regs)}
		for i, r := range regs {
			v, _ := strconv.ParseInt(r[8], 10, 64)
			l, _ := strconv.ParseInt(r[10], 10, 64)
			a.Bruto += v
			a.Liquido += l
			b.WriteString(aspas(append(r, fmt.Sprintf("%06d", i+2))) + "\n")
		}
		b.WriteString(aspas([]string{"9", fmt.Sprintf("%06d", len(regs)+2)}) + "\n")
		a.Nome = fmt.Sprintf("stone-1503-recebimento-%s-%s-%s.csv", d[6:8], d[4:6], d[0:4])
		a.Conteudo = []byte(b.String())
		res.Arquivos = append(res.Arquivos, a)
	}

	res.NaoConvert = len(conf)
	if len(conf) > 0 {
		var b bytes.Buffer
		b.Write([]byte{0xEF, 0xBB, 0xBF})
		w := csv.NewWriter(&b)
		w.Comma = ';'
		w.UseCRLF = true
		w.Write([]string{"MOTIVO", "CATEGORIA", "DATA DA VENDA", "VENCIMENTO", "BANDEIRA", "PRODUTO",
			"STONE ID", "QTD PARCELAS", "PARCELA", "VALOR BRUTO", "VALOR LIQUIDO", "STATUS"})
		// O que exige acao (estorno no WinThor) vai para o topo da conferencia.
		sort.SliceStable(conf, func(i, j int) bool {
			return conf[i][0] == motivoCancelOutro && conf[j][0] != motivoCancelOutro
		})
		w.WriteAll(conf)
		w.Flush()
		res.confConteudo = b.Bytes()
	}
	return res, nil
}

// Resumo devolve o texto curto mostrado na janela (considera so os arquivos salvos).
func (r *Resultado) Resumo() string {
	var b strings.Builder
	salvos := 0
	for _, a := range r.Arquivos {
		if a.Caminho != "" {
			salvos++
		}
	}
	switch {
	case len(r.Arquivos) == 0:
		b.WriteString("Nenhuma linha convertida.\nO CSV não tem vendas com status \"Pago\".")
	case salvos == 0:
		b.WriteString("Cancelado: nenhum arquivo foi salvo.\nArraste o CSV de novo para converter.")
	default:
		p, bruto, liq := r.Totais()
		if len(r.Arquivos) == 1 {
			a := r.Arquivos[0]
			fmt.Fprintf(&b, "Concluído: recebimento de %s\n%s\n", dataBR(a.Data), filepath.Base(a.Caminho))
		} else {
			fmt.Fprintf(&b, "ATENÇÃO: %d datas de recebimento no CSV\n%d de %d arquivos salvos (%s a %s).\n",
				len(r.Arquivos), salvos, len(r.Arquivos), dataBR(r.Arquivos[0].Data), dataBR(r.Arquivos[len(r.Arquivos)-1].Data))
		}
		fmt.Fprintf(&b, "%d parcelas  |  Bruto %s\nTaxa %s  |  Líquido %s", p, reais(bruto), reais(bruto-liq), reais(liq))
	}
	if r.NaoConvert > 0 {
		fmt.Fprintf(&b, "\n%d linhas não convertidas (ver conferência)", r.NaoConvert)
	}
	if r.CancelOutroPeriodo > 0 {
		fmt.Fprintf(&b, "\n⚠ %d cancelamento(s) de venda já baixada: estornar no WinThor (ver topo da conferência)",
			r.CancelOutroPeriodo)
	}
	return b.String()
}

// AvisoDatas devolve o texto do aviso quando ha mais de uma data de recebimento.
func (r *Resultado) AvisoDatas() string {
	if len(r.Arquivos) < 2 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "O CSV da Stone tinha %d datas de recebimento.\nFoi gerado um arquivo para cada data:\n\n", len(r.Arquivos))
	for _, a := range r.Arquivos {
		fmt.Fprintf(&b, "%s   %d parcelas   líquido %s\n", dataBR(a.Data), a.Parcelas, reais(a.Liquido))
	}
	b.WriteString("\nVocê vai escolher onde salvar cada arquivo.\nImporte e baixe um arquivo de cada vez na rotina 1503.")
	return b.String()
}
