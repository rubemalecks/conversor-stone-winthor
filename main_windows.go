//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	appTitulo = "Conversor Stone → 1503"
	appSub    = "Recebimentos da Stone para a rotina 1503 do WinThor"
	autor     = "Autor: Rubem · RBTech Soluções"
	gitTexto  = "github.com/rubemalecks"
	gitURL    = "https://github.com/rubemalecks"

	cliW, cliH = 400, 276
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")

	pRegisterClassExW   = user32.NewProc("RegisterClassExW")
	pCreateWindowExW    = user32.NewProc("CreateWindowExW")
	pDefWindowProcW     = user32.NewProc("DefWindowProcW")
	pGetMessageW        = user32.NewProc("GetMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessageW   = user32.NewProc("DispatchMessageW")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pShowWindow         = user32.NewProc("ShowWindow")
	pUpdateWindow       = user32.NewProc("UpdateWindow")
	pBeginPaint         = user32.NewProc("BeginPaint")
	pEndPaint           = user32.NewProc("EndPaint")
	pFillRect           = user32.NewProc("FillRect")
	pDrawTextW          = user32.NewProc("DrawTextW")
	pInvalidateRect     = user32.NewProc("InvalidateRect")
	pLoadCursorW        = user32.NewProc("LoadCursorW")
	pSetCursor          = user32.NewProc("SetCursor")
	pLoadImageW         = user32.NewProc("LoadImageW")
	pDrawIconEx         = user32.NewProc("DrawIconEx")
	pMessageBoxW        = user32.NewProc("MessageBoxW")
	pAdjustWindowRectEx = user32.NewProc("AdjustWindowRectEx")
	pGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	pGetCursorPos       = user32.NewProc("GetCursorPos")
	pScreenToClient     = user32.NewProc("ScreenToClient")

	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pCreatePen        = gdi32.NewProc("CreatePen")
	pCreateFontW      = gdi32.NewProc("CreateFontW")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pRectangle        = gdi32.NewProc("Rectangle")
	pGetStockObject   = gdi32.NewProc("GetStockObject")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	pDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
	pDragQueryFileW  = shell32.NewProc("DragQueryFileW")
	pDragFinish      = shell32.NewProc("DragFinish")
	pShellExecuteW   = shell32.NewProc("ShellExecuteW")

	pSetTimer         = user32.NewProc("SetTimer")
	pKillTimer        = user32.NewProc("KillTimer")
	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	pGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
)

const (
	wmDestroy    = 0x0002
	wmPaint      = 0x000F
	wmEraseBkgnd = 0x0014
	wmSetCursor  = 0x0020
	wmTimer      = 0x0113
	wmLButtonUp  = 0x0202

	timerAbrir    = 1
	segundosAbrir = 3 // espera antes de abrir a pasta automaticamente
	wmDropFiles   = 0x0233

	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsExAcceptFiles = 0x00000010

	dtCenter       = 0x1
	dtRight        = 0x2
	dtVCenter      = 0x4
	dtWordBreak    = 0x10
	dtSingleLine   = 0x20
	dtNoPrefix     = 0x800
	dtPathEllipsis = 0x4000
	dtEndEllipsis  = 0x8000

	ofnOverwritePrompt = 0x00000002
	ofnHideReadOnly    = 0x00000004
	ofnNoChangeDir     = 0x00000008
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnExplorer        = 0x00080000

	mbYesNo       = 0x04
	mbIconError   = 0x10
	mbIconWarning = 0x30
	mbIconInfo    = 0x40
	idYes         = 6

	corAzul   = 0x009C5F0D // RGB(13,95,156)
	corFundo  = 0x00FAF8F6
	corBranco = 0x00FFFFFF
	corTexto  = 0x00222222
	corCinza  = 0x00776655
	corBorda  = 0x00C9A97F
	corAlerta = 0x00005B9A // RGB(154,91,0)
	corLink   = 0x00FAEBD7
)

type rect struct{ left, top, right, bottom int32 }
type point struct{ x, y int32 }

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
	private uint32
}

type paintStruct struct {
	hdc       uintptr
	erase     int32
	rcPaint   rect
	restore   int32
	incUpdate int32
	reserved  [32]byte
}

type openFileName struct {
	structSize    uint32
	owner         uintptr
	instance      uintptr
	filter        *uint16
	customFilter  *uint16
	maxCustFilter uint32
	filterIndex   uint32
	file          *uint16
	maxFile       uint32
	fileTitle     *uint16
	maxFileTitle  uint32
	initialDir    *uint16
	title         *uint16
	flags         uint32
	fileOffset    uint16
	fileExtension uint16
	defExt        *uint16
	custData      uintptr
	hook          uintptr
	templateName  *uint16
	reserved1     uintptr
	reserved2     uint32
	flagsEx       uint32
}

// Areas da janela (coordenadas de cliente).
var (
	rcZona   = rect{14, 62, cliW - 14, 216}
	rcStatus = rect{14, 221, cliW - 14, 245}
	rcRodape = rect{0, cliH - 28, cliW, cliH}
	rcLink   = rect{cliW - 170, cliH - 28, cliW - 10, cliH}
)

// Estado mostrado na janela.
var (
	hwndMain    uintptr
	hIcone      uintptr
	textoZona   string
	alerta      bool
	pastaSaida  string
	ultimaPasta string // ultima pasta escolhida para salvar

	confirmacao string // texto da faixa azul de concluido ("" = sem faixa)
	pastaAbrir  string // pasta que sera aberta ao fim da contagem
	contagem    int    // segundos restantes para abrir a pasta

	fTitulo, fNormal, fPequena, fNegrito, fLink uintptr
)

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func fonte(altura int32, peso int32, sublinhado uintptr) uintptr {
	h, _, _ := pCreateFontW.Call(uintptr(altura), 0, 0, 0, uintptr(peso), 0, sublinhado, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(u16("Segoe UI"))))
	return h
}

func dentro(r rect, x, y int32) bool {
	return x >= r.left && x < r.right && y >= r.top && y < r.bottom
}

func caixa(texto string, flags uintptr) uintptr {
	r, _, _ := pMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(u16(texto))),
		uintptr(unsafe.Pointer(u16(appTitulo))), flags)
	return r
}

// confirmarGerados mostra a faixa azul de concluido e agenda a abertura da pasta.
func confirmarGerados(res *Resultado) {
	n := 0
	for _, a := range res.Arquivos {
		if a.Caminho != "" {
			n++
		}
	}
	if n == 0 || res.Pasta == "" {
		return
	}
	confirmacao = "✔ Concluído! Arquivo gerado"
	if n > 1 {
		confirmacao = fmt.Sprintf("✔ Concluído! %d arquivos gerados", n)
	}
	pastaAbrir = res.Pasta
	contagem = segundosAbrir
	pSetTimer.Call(hwndMain, timerAbrir, 1000, 0)
	repintar()
}

// pararContagem cancela a abertura automatica pendente.
func pararContagem() {
	pKillTimer.Call(hwndMain, timerAbrir)
	contagem = 0
}

func abrir(alvo string) {
	pShellExecuteW.Call(hwndMain, uintptr(unsafe.Pointer(u16("open"))),
		uintptr(unsafe.Pointer(u16(alvo))), 0, 0, 1)
}

func repintar() {
	pInvalidateRect.Call(hwndMain, 0, 1)
}

func converterArquivos(arquivos []string) {
	// Nova conversao: some a faixa de concluido anterior.
	pararContagem()
	confirmacao = ""
	repintar()
	var resumos []string
	for _, a := range arquivos {
		if strings.ToLower(filepath.Ext(a)) != ".csv" {
			caixa("Arraste o arquivo CSV de recebimentos da Stone.\n\nArquivo recebido: "+filepath.Base(a), mbIconError)
			continue
		}
		res, err := Converter(a)
		if err != nil {
			caixa(filepath.Base(a)+"\n\n"+err.Error(), mbIconError)
			continue
		}
		alerta = len(res.Arquivos) != 1 || res.CancelOutroPeriodo > 0
		if aviso := res.AvisoDatas(); aviso != "" {
			caixa(aviso, mbIconWarning)
		}

		// Pergunta onde salvar cada arquivo, ja com o nome sugerido (data do recebimento).
		pasta := ultimaPasta
		if pasta == "" {
			pasta = filepath.Dir(a)
		}
		for i := range res.Arquivos {
			arq := &res.Arquivos[i]
			destino, ok := salvarComo(arq.Nome, pasta, "Salvar arquivo da 1503 - recebimento de "+dataBR(arq.Data))
			if !ok {
				continue
			}
			if err := arq.Gravar(destino); err != nil {
				caixa(err.Error(), mbIconError)
				continue
			}
			pasta = filepath.Dir(destino)
			ultimaPasta = pasta
			res.Pasta = pasta
		}

		// A conferencia vai para a mesma pasta dos arquivos salvos.
		if res.NaoConvert > 0 {
			destConf := res.Pasta
			if destConf == "" {
				destConf = filepath.Dir(a)
			}
			if err := res.GravarConferencia(destConf); err != nil {
				caixa(err.Error(), mbIconError)
			} else if res.Pasta == "" {
				res.Pasta = destConf
			}
		}
		if res.Pasta != "" {
			pastaSaida = res.Pasta
		}
		resumos = append(resumos, res.Resumo())
		textoZona = strings.Join(resumos, "\n")
		repintar()
		confirmarGerados(res)
	}
}

// utf16z converte para UTF-16 terminado em NUL, aceitando NULs no meio
// (necessario para os filtros dos dialogos de arquivo).
func utf16z(s string) []uint16 {
	return append(utf16.Encode([]rune(s)), 0)
}

const filtroCSV = "Arquivo CSV (*.csv)\x00*.csv\x00Todos os arquivos\x00*.*\x00"

func escolherArquivo() {
	buf := make([]uint16, 4096)
	filtro := utf16z(filtroCSV)
	ofn := openFileName{
		owner:   hwndMain,
		filter:  &filtro[0],
		file:    &buf[0],
		maxFile: uint32(len(buf)),
		title:   u16("Selecione o CSV de recebimentos da Stone"),
		flags:   ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnHideReadOnly | ofnNoChangeDir,
	}
	ofn.structSize = uint32(unsafe.Sizeof(ofn))
	if r, _, _ := pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn))); r != 0 {
		converterArquivos([]string{syscall.UTF16ToString(buf)})
	}
}

// salvarComo abre o "Salvar como" com o nome sugerido e devolve o caminho escolhido.
func salvarComo(nome, pasta, titulo string) (string, bool) {
	buf := make([]uint16, 4096)
	copy(buf, utf16.Encode([]rune(nome)))
	filtro := utf16z(filtroCSV)
	ofn := openFileName{
		owner:      hwndMain,
		filter:     &filtro[0],
		file:       &buf[0],
		maxFile:    uint32(len(buf)),
		initialDir: u16(pasta),
		title:      u16(titulo),
		defExt:     u16("csv"),
		flags:      ofnExplorer | ofnOverwritePrompt | ofnPathMustExist | ofnHideReadOnly | ofnNoChangeDir,
	}
	ofn.structSize = uint32(unsafe.Sizeof(ofn))
	r, _, _ := pGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filtro)
	runtime.KeepAlive(buf)
	if r == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func texto(hdc uintptr, s string, r rect, f uintptr, cor uintptr, flags uintptr) {
	pSelectObject.Call(hdc, f)
	pSetTextColor.Call(hdc, cor)
	p := syscall.StringToUTF16(s)
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)-1),
		uintptr(unsafe.Pointer(&r)), flags|dtNoPrefix)
}

func preencher(hdc uintptr, r rect, cor uintptr) {
	b, _, _ := pCreateSolidBrush.Call(cor)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), b)
	pDeleteObject.Call(b)
}

func pintar(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	pSetBkMode.Call(hdc, 1) // TRANSPARENT

	preencher(hdc, rect{0, 0, cliW, cliH}, corFundo)

	// Cabecalho
	if hIcone != 0 {
		pDrawIconEx.Call(hdc, 14, 10, hIcone, 40, 40, 0, 0, 3 /*DI_NORMAL*/)
	}
	texto(hdc, appTitulo, rect{64, 9, cliW - 10, 33}, fTitulo, corAzul, dtSingleLine)
	texto(hdc, appSub, rect{64, 33, cliW - 10, 52}, fPequena, corCinza, dtSingleLine|dtEndEllipsis)

	// Area de arrastar
	preencher(hdc, rcZona, corBranco)
	pen, _, _ := pCreatePen.Call(1 /*PS_DASH*/, 1, corBorda)
	oco, _, _ := pGetStockObject.Call(5 /*NULL_BRUSH*/)
	penAnt, _, _ := pSelectObject.Call(hdc, pen)
	brAnt, _, _ := pSelectObject.Call(hdc, oco)
	pRectangle.Call(hdc, uintptr(rcZona.left), uintptr(rcZona.top), uintptr(rcZona.right), uintptr(rcZona.bottom))
	pSelectObject.Call(hdc, penAnt)
	pSelectObject.Call(hdc, brAnt)
	pDeleteObject.Call(pen)

	if textoZona == "" {
		meio := (rcZona.top + rcZona.bottom) / 2
		texto(hdc, "Arraste o CSV da Stone aqui", rect{rcZona.left, meio - 24, rcZona.right, meio},
			fNormal, corAzul, dtCenter|dtSingleLine)
		texto(hdc, "ou clique para selecionar", rect{rcZona.left, meio + 2, rcZona.right, meio + 22},
			fPequena, corCinza, dtCenter|dtSingleLine)
	} else {
		linhas := strings.SplitN(textoZona, "\n", 2)
		cor := uintptr(corAzul)
		if alerta {
			cor = corAlerta
		}
		texto(hdc, linhas[0], rect{rcZona.left + 12, rcZona.top + 10, rcZona.right - 10, rcZona.top + 30},
			fNegrito, cor, dtSingleLine|dtEndEllipsis)
		if len(linhas) > 1 {
			texto(hdc, linhas[1], rect{rcZona.left + 12, rcZona.top + 34, rcZona.right - 10, rcZona.bottom - 6},
				fPequena, corTexto, dtWordBreak)
		}
	}

	// Linha de status
	rcTxt := rect{rcStatus.left + 2, rcStatus.top, rcStatus.right - 2, rcStatus.bottom}
	if confirmacao != "" {
		// Faixa azul de concluido, com a contagem para abrir a pasta.
		preencher(hdc, rcStatus, corAzul)
		msg := confirmacao + " · pasta aberta"
		if contagem > 0 {
			msg = fmt.Sprintf("%s · abrindo em %ds", confirmacao, contagem)
		}
		texto(hdc, msg, rect{rcStatus.left + 10, rcStatus.top, rcStatus.right - 8, rcStatus.bottom},
			fNegrito, corBranco, dtSingleLine|dtVCenter|dtEndEllipsis)
	} else if pastaSaida != "" {
		texto(hdc, "Abrir pasta: "+pastaSaida, rcTxt, fLink, corAzul, dtSingleLine|dtVCenter|dtPathEllipsis)
	} else {
		texto(hdc, "Você escolhe onde salvar; o nome leva a data do recebimento.", rcTxt,
			fPequena, corCinza, dtSingleLine|dtVCenter|dtEndEllipsis)
	}

	// Rodape
	preencher(hdc, rcRodape, corAzul)
	texto(hdc, autor, rect{10, rcRodape.top, cliW - 175, rcRodape.bottom}, fPequena, corBranco,
		dtSingleLine|dtVCenter|dtEndEllipsis)
	texto(hdc, gitTexto, rcLink, fLink, corLink, dtSingleLine|dtVCenter|dtRight)

	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func wndProc(hwnd, m, wParam, lParam uintptr) uintptr {
	switch m {
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		pintar(hwnd)
		return 0
	case wmSetCursor:
		if lParam&0xFFFF == 1 { // HTCLIENT
			var p point
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
			pScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&p)))
			if dentro(rcZona, p.x, p.y) || dentro(rcLink, p.x, p.y) ||
				(pastaSaida != "" && dentro(rcStatus, p.x, p.y)) {
				mao, _, _ := pLoadCursorW.Call(0, 32649) // IDC_HAND
				pSetCursor.Call(mao)
				return 1
			}
		}
	case wmTimer:
		if wParam == timerAbrir {
			contagem--
			if contagem <= 0 {
				pararContagem()
				abrir(pastaAbrir)
			}
			repintar()
		}
		return 0
	case wmLButtonUp:
		x, y := int32(int16(lParam&0xFFFF)), int32(int16((lParam>>16)&0xFFFF))
		switch {
		case dentro(rcLink, x, y):
			abrir(gitURL)
		case pastaSaida != "" && dentro(rcStatus, x, y):
			if contagem > 0 { // clicou na faixa: abre ja, sem esperar
				pararContagem()
				repintar()
			}
			abrir(pastaSaida)
		case dentro(rcZona, x, y):
			escolherArquivo()
		}
		return 0
	case wmDropFiles:
		n, _, _ := pDragQueryFileW.Call(wParam, 0xFFFFFFFF, 0, 0)
		var arquivos []string
		for i := uintptr(0); i < n; i++ {
			buf := make([]uint16, 4096)
			pDragQueryFileW.Call(wParam, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			arquivos = append(arquivos, syscall.UTF16ToString(buf))
		}
		pDragFinish.Call(wParam)
		converterArquivos(arquivos)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, m, wParam, lParam)
	return r
}

func main() {
	runtime.LockOSThread()

	inst, _, _ := pGetModuleHandleW.Call(0)
	hIcone, _, _ = pLoadImageW.Call(inst, 1, 1 /*IMAGE_ICON*/, 48, 48, 0)
	iconeGrande, _, _ := pLoadImageW.Call(inst, 1, 1, 32, 32, 0)
	iconePeq, _, _ := pLoadImageW.Call(inst, 1, 1, 16, 16, 0)
	seta, _, _ := pLoadCursorW.Call(0, 32512) // IDC_ARROW

	fTitulo = fonte(-19, 700, 0)
	fNormal = fonte(-16, 400, 0)
	fNegrito = fonte(-14, 700, 0)
	fPequena = fonte(-13, 400, 0)
	fLink = fonte(-13, 400, 1)

	classe := u16("RBTechConversorStone1503")
	wc := wndClassEx{
		wndProc:   syscall.NewCallback(wndProc),
		instance:  inst,
		icon:      iconeGrande,
		iconSm:    iconePeq,
		cursor:    seta,
		className: classe,
	}
	wc.size = uint32(unsafe.Sizeof(wc))
	if r, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return
	}

	estilo := uintptr(wsCaption | wsSysMenu | wsMinimizeBox)
	r := rect{0, 0, cliW, cliH}
	pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), estilo, 0, wsExAcceptFiles)
	w, h := r.right-r.left, r.bottom-r.top
	sw, _, _ := pGetSystemMetrics.Call(0)
	sh, _, _ := pGetSystemMetrics.Call(1)
	x, y := (int32(sw)-w)/2, (int32(sh)-h)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	hwndMain, _, _ = pCreateWindowExW.Call(wsExAcceptFiles, uintptr(unsafe.Pointer(classe)),
		uintptr(unsafe.Pointer(u16(appTitulo))), estilo,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, inst, 0)
	if hwndMain == 0 {
		return
	}
	pDragAcceptFiles.Call(hwndMain, 1)
	pShowWindow.Call(hwndMain, 1)
	pUpdateWindow.Call(hwndMain)

	// Arquivos soltos em cima do icone do .exe chegam como argumentos.
	if len(os.Args) > 1 {
		converterArquivos(os.Args[1:])
	}

	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
