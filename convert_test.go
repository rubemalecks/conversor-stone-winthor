package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCentavos(t *testing.T) {
	casos := map[string]int64{
		"1.524,330000": 152433,
		"1.490,115000": 149012, // arredonda para cima
		"97,504999":    9750,
		"0,995":        100,
		"-50,000000":   -5000,
		"10":           1000,
	}
	for in, want := range casos {
		got, err := centavos(in)
		if err != nil || got != want {
			t.Errorf("centavos(%q) = %d, %v; quer %d", in, got, err, want)
		}
	}
	if _, err := centavos("abc"); err == nil {
		t.Error("centavos(\"abc\") deveria dar erro")
	}
}

func TestReais(t *testing.T) {
	casos := map[int64]string{0: "0,00", 5: "0,05", 152433: "1.524,33", 123456789: "1.234.567,89", -5000: "-50,00"}
	for in, want := range casos {
		if got := reais(in); got != want {
			t.Errorf("reais(%d) = %q; quer %q", in, got, want)
		}
	}
}

func TestConverter(t *testing.T) {
	csv := "STONECODE;CATEGORIA;DATA DA VENDA;DATA DE VENCIMENTO;BANDEIRA;PRODUTO;STONE ID;QTD DE PARCELAS;Nº DA PARCELA;VALOR BRUTO;VALOR LÍQUIDO;ÚLTIMO STATUS\n" +
		"123456;Venda;30/10/2026 14:53:22;30/11/2026;Mastercard;Crédito;111;1;1;1.524,330000;1.490,115000;Pago\n" +
		"123456;Venda;30/10/2026 15:00:00;02/12/2026;Visa;Crédito;222;3;2;100,000000;97,500000;Pago\n" +
		"123456;Venda;30/10/2026 16:00:00;01/12/2026;Elo;Débito;333;1;1;50,000000;49,000000;Pago\n" +
		"123456;Cancelamento;30/10/2026 17:00:00;01/12/2026;Elo;Débito;333;1;1;-50,000000;-49,000000;Pago\n" +
		"123456;Venda;30/10/2026 18:00:00;01/12/2026;BandeiraInventada;Crédito;444;1;1;10,000000;9,000000;Pago\n" +
		"123456;Venda;30/10/2026 18:00:00;02/12/2026;Visa;Crédito;555;1;1;10,000000;9,000000;Previsto\n"
	dir := t.TempDir()
	origem := filepath.Join(dir, "stone.csv")
	if err := os.WriteFile(origem, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Converter(origem)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arquivos) != 2 {
		t.Fatalf("esperava 2 arquivos (2 datas), veio %d", len(res.Arquivos))
	}
	if res.Arquivos[0].Nome != "stone-1503-recebimento-30-11-2026.csv" ||
		res.Arquivos[1].Nome != "stone-1503-recebimento-02-12-2026.csv" {
		t.Errorf("nomes inesperados: %s, %s", res.Arquivos[0].Nome, res.Arquivos[1].Nome)
	}
	if res.NaoConvert != 4 {
		t.Errorf("esperava 4 linhas na conferencia, veio %d (%v)", res.NaoConvert, res.Motivos)
	}

	// Nada e gravado antes de pedir.
	if m, _ := filepath.Glob(filepath.Join(dir, "stone-1503-*")); len(m) != 0 {
		t.Errorf("Converter nao deveria gravar arquivos: %v", m)
	}

	if err := res.GravarTodos(dir); err != nil {
		t.Fatal(err)
	}
	conteudo, err := os.ReadFile(filepath.Join(dir, "stone-1503-recebimento-30-11-2026.csv"))
	if err != nil {
		t.Fatal(err)
	}
	linhas := strings.Split(strings.TrimSpace(string(conteudo)), "\n")
	if len(linhas) != 3 {
		t.Fatalf("esperava cabecalho + 1 registro + trailer, veio %d linhas", len(linhas))
	}
	reg := strings.Split(strings.Trim(linhas[1], `"`), `";"`)
	confere := map[int]string{0: "10", 2: "000000000123456", 3: "20261030", 6: "111", 8: "152433",
		10: "149012", 12: "20261130", 21: "3421", 22: "224", 26: "0007", 28: "145322"}
	for i, want := range confere {
		if reg[i] != want {
			t.Errorf("campo %d = %q; quer %q", i+1, reg[i], want)
		}
	}
	if linhas[2] != `"9";"000003"` {
		t.Errorf("trailer = %s", linhas[2])
	}
	if _, err := os.Stat(filepath.Join(dir, "stone-conferencia.csv")); err != nil {
		t.Errorf("conferencia nao gravada: %v", err)
	}
}

// Amex, debito (campo 16 = D) e cancelamento de venda de outro periodo.
func TestBandeirasProdutoECancelOutroPeriodo(t *testing.T) {
	csv := "STONECODE;CATEGORIA;DATA DA VENDA;DATA DE VENCIMENTO;BANDEIRA;PRODUTO;STONE ID;QTD DE PARCELAS;Nº DA PARCELA;VALOR BRUTO;VALOR LÍQUIDO;ÚLTIMO STATUS\n" +
		"1;Venda;30/10/2026 10:00:00;04/11/2026;AmericanExpress;Credito;A1;1;1;105,280000;102,963840;Pago\n" +
		"1;Venda;30/10/2026 10:00:00;04/11/2026;Visa;Debito Pre-pago;D1;1;1;10,000000;9,900000;Pago\n" +
		"1;Venda;30/10/2026 10:00:00;04/11/2026;Elo;Credito Pre-pago;C1;1;1;10,000000;9,800000;Pago\n" +
		"1;Cancelamento;01/11/2026 10:00:00;04/11/2026;MasterCard;Credito;X9;2;1;-296,690000;-290,311165;Pago\n" +
		"1;Balanceamento de Saldo;01/11/2026 10:00:00;04/11/2026;MasterCard;Credito;;1;1;290,310000;290,310000;Pago\n"
	dir := t.TempDir()
	origem := filepath.Join(dir, "nov.csv")
	os.WriteFile(origem, []byte(csv), 0o644)
	res, err := Converter(origem)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Arquivos) != 1 || res.Arquivos[0].Parcelas != 3 {
		t.Fatalf("esperava 1 arquivo com 3 parcelas: %+v", res.Arquivos)
	}
	quer := map[string][2]string{"A1": {"C", "0001"}, "D1": {"D", "0008"}, "C1": {"C", "0004"}}
	for _, l := range strings.Split(strings.TrimSpace(string(res.Arquivos[0].Conteudo)), "\n") {
		reg := strings.Split(strings.Trim(l, `"`), `";"`)
		if reg[0] != "10" {
			continue
		}
		q := quer[reg[6]]
		if reg[15] != q[0] || reg[26] != q[1] || reg[17] != "369" {
			t.Errorf("%s: produto=%s bandeira=%s rede=%s; quer %v e rede 369", reg[6], reg[15], reg[26], reg[17], q)
		}
	}
	if res.CancelOutroPeriodo != 1 {
		t.Errorf("esperava 1 cancelamento de outro período, veio %d", res.CancelOutroPeriodo)
	}
	if !strings.HasPrefix(strings.SplitN(string(res.confConteudo), "\r\n", 3)[1], "\"ATENÇÃO") &&
		!strings.HasPrefix(strings.SplitN(string(res.confConteudo), "\r\n", 3)[1], "ATENÇÃO") {
		t.Errorf("o cancelamento de outro período deveria estar no topo da conferência:\n%s", res.confConteudo)
	}
	if !strings.Contains(res.Resumo(), "estornar no WinThor") {
		t.Errorf("resumo sem o aviso de estorno: %s", res.Resumo())
	}
}

func TestCodigoBandeira(t *testing.T) {
	casos := []struct{ bandeira, produto, quer string }{
		{"MasterCard", "C", "0007"}, {"MasterCard", "D", "0007"},
		{"Visa", "D", "0008"}, {"Elo", "C", "0004"},
		{"AmericanExpress", "C", "0001"}, {"American Express", "C", "0001"}, {"Amex", "C", "0001"},
		{"Hipercard", "C", "0005"}, {"Hiper", "C", "0031"}, {"Hiper", "D", "0022"},
		{"Banescard", "C", "0029"}, {"Banescard", "D", "0019"},
		{"Siscred", "C", "0011"}, {"Siscred", "D", "0012"},
		{"Diners Club", "C", "0003"}, {"Cabal", "D", "0002"}, {"JCB", "C", "0030"},
		{"Maestro", "D", "0006"}, {"Visa Electron", "D", "0009"}, {"UnionPay", "C", "0015"},
		{"Sorocred", "C", "0013"}, {"CredZ", "C", "0021"}, {"Ourocard", "C", "0035"},
	}
	for _, c := range casos {
		if got, ok := codigoBandeira(c.bandeira, c.produto); !ok || got != c.quer {
			t.Errorf("codigoBandeira(%q, %q) = %q, %v; quer %q", c.bandeira, c.produto, got, ok, c.quer)
		}
	}
	if _, ok := codigoBandeira("BandeiraInventada", "C"); ok {
		t.Error("bandeira desconhecida deveria falhar")
	}
	// Todo codigo usado precisa existir na Tabela II (0001-0035).
	for nome, b := range bandeiras {
		for _, c := range []string{b.credito, b.debito} {
			if len(c) != 4 || c < "0001" || c > "0035" {
				t.Errorf("%s: código %q fora da Tabela II", nome, c)
			}
		}
	}
}

func TestColunasFaltando(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "outro.csv")
	os.WriteFile(p, []byte("A;B\n1;2\n"), 0o644)
	if _, err := Converter(p); err == nil || !strings.Contains(err.Error(), "Colunas não encontradas") {
		t.Errorf("esperava erro de colunas, veio %v", err)
	}
}
