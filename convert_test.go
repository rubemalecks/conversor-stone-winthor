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
		"123456;Venda;30/10/2026 18:00:00;01/12/2026;Amex;Crédito;444;1;1;10,000000;9,000000;Pago\n" +
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

func TestColunasFaltando(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "outro.csv")
	os.WriteFile(p, []byte("A;B\n1;2\n"), 0o644)
	if _, err := Converter(p); err == nil || !strings.Contains(err.Error(), "Colunas não encontradas") {
		t.Errorf("esperava erro de colunas, veio %v", err)
	}
}
