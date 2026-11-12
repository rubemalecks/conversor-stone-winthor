# Conversor Stone → 1503

Converte o **CSV de recebimentos da Stone** para o arquivo de retorno no **padrão 4.1 (layout 1 – SOFTWARE EXPRESS)** que a **rotina 1503 do WinThor** importa para dar baixa nos títulos de cartão.

Um único `.exe` para Windows, sem instalação e sem dependências: arraste o CSV, escolha onde salvar e importe na 1503.

![Tela do programa](docs/tela.png)

## Download

Baixe o `ConversorStone1503.exe` na página de [Releases](../../releases/latest) e execute. Não precisa instalar nada.

## Como usar

1. No portal da Stone, exporte o relatório de **recebimentos** em CSV.
2. Abra o `ConversorStone1503.exe` e **arraste o CSV** para a janela. Você também pode clicar na área pontilhada para selecionar o arquivo, ou soltar o CSV em cima do ícone do `.exe`.
3. Na janela **Salvar como**, confirme onde salvar. O nome já vem preenchido com a data do recebimento, por exemplo `stone-1503-recebimento-30-11-2026.csv`.
4. Aparece a faixa azul **"✔ Concluído!"** e, depois de 3 segundos, a pasta de destino abre sozinha.
5. Na **rotina 1503** do WinThor, importe o arquivo e faça a baixa.

> Se o CSV tiver **mais de uma data de recebimento**, o programa avisa e gera **um arquivo por data**. Ele pergunta onde salvar cada um. Importe e baixe um arquivo de cada vez.

## O que é convertido

Entra no arquivo da 1503 cada linha do CSV que:

- tenha categoria **Venda**;
- tenha último status **Pago**;
- tenha **Stone ID**;
- seja de uma bandeira com código conhecido (**Mastercard `0007`, Visa `0008`, Elo `0004`**);
- tenha valor bruto positivo;
- **não** tenha cancelamento do mesmo Stone ID, mesma parcela e mesmo valor.

O resto vai para o arquivo **`<nome-do-csv>-conferencia.csv`**, salvo na mesma pasta. Ele traz o **motivo** de cada linha ter ficado de fora.

Para cada parcela, o programa calcula a taxa (bruto − líquido) e o percentual da taxa, e coloca o **Stone ID no campo NSU**. A rotina 1503 encontra o título comparando esse campo com `PCPREST.NSUHOST`.

## Antes de usar em outra empresa

O programa **não tem dados de nenhuma empresa gravados no código**: nenhum CNPJ, filial ou conta. O código do estabelecimento (STONECODE) vem do próprio CSV. Mesmo assim, confira estes pontos:

| Ponto | Situação |
|---|---|
| **NSU = Stone ID** | Só dá baixa se os títulos de cartão estiverem gravados no WinThor com o Stone ID em `PCPREST.NSUHOST`. Isso depende da integração PDV/TEF de cada empresa. |
| Campo fixo `369` | Provavelmente o código da adquirente no layout. Confirme no manual do layout 1 da 1503. |
| Campo fixo `C` | Sai `C` (crédito) em **todas** as linhas, inclusive débito. Confirme se a 1503 diferencia. |
| Outras bandeiras | Hipercard, Amex, Cabal etc. não têm código mapeado e vão para a conferência. Para incluí-las, acrescente os códigos em `bandeiras`, no `convert.go`. |
| Formato dos valores | O programa espera vírgula decimal (`1.524,33`), que é como a Stone exporta hoje. |

## Compilar

Requer o [Go 1.24+](https://go.dev/dl/).

```powershell
.\build.ps1            # roda go vet + testes e gera dist\ConversorStone1503.exe
.\build.ps1 -SemTestes # só compila
go test ./...          # só os testes
```

O ícone do executável fica em `rsrc_windows_amd64.syso`, gerado a partir de `assets/icone.ico`. Para trocar o ícone, gere um novo `.syso`, por exemplo com o [akavel/rsrc](https://github.com/akavel/rsrc):

```powershell
rsrc -ico assets\icone.ico -o rsrc_windows_amd64.syso
```

Fora do Windows, o mesmo código compila como **linha de comando** (`go run . arquivo.csv`). Útil para testar a conversão: os arquivos são gravados na pasta do CSV.

## Estrutura

```
convert.go               leitura do CSV da Stone e geração do layout 4.1 (sem interface)
convert_test.go          testes da conversão
main_windows.go          janela Win32 (arrastar e soltar, Salvar como, faixa de concluído)
main_other.go            versão linha de comando para outros sistemas
rsrc_windows_amd64.syso  ícone embutido no .exe
assets/icone.ico         ícone original
build.ps1                script de compilação
docs/                    imagens do README
```

## Autor

Rubem · RBTech Soluções — [github.com/rubemalecks](https://github.com/rubemalecks)
