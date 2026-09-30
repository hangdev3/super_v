package clipboard

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

type SystemClipboard struct {
	imageDirectory string
	output         clipboardCommandOutput
}

// ErrClipboardNotText means that the current selection is an image or another
// non-text MIME type. The monitor intentionally ignores this state instead of
// converting image data into a bogus text history entry.
var ErrClipboardNotText = errors.New("clipboard does not contain text")

func NewSystemClipboard() (*SystemClipboard, error) {
	if clipboard.Unsupported {
		return nil, errors.New("clipboard indisponivel: instale xclip ou xsel no X11")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("obter diretorio pessoal para imagens do clipboard: %w", err)
	}

	return &SystemClipboard{
		imageDirectory: filepath.Join(home, "Imagens", "Captures"),
		output:         runClipboardCommand,
	}, nil
}

func (c SystemClipboard) ReadText() (string, error) {
	// atotto's generic xclip command does not request a MIME target. When the
	// selection is an image, that can result in binary/adapter metadata being
	// interpreted as text. Inspect X11 targets first and only request a text
	// target. Wayland keeps atotto's native wl-paste path for now.
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		if xclip, err := exec.LookPath("xclip"); err == nil {
			text, err := readTextWithXclip(xclip, c.commandOutput())
			if err != nil {
				return "", fmt.Errorf("ler clipboard: %w", err)
			}
			return text, nil
		}
		if xsel, err := exec.LookPath("xsel"); err == nil {
			text, err := readTextWithXsel(xsel, c.commandOutput())
			if err != nil {
				return "", fmt.Errorf("ler clipboard: %w", err)
			}
			return text, nil
		}
	}

	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("ler clipboard: %w", err)
	}

	return text, nil
}

// ReadImage reads the current X11 image selection into the local captures
// directory. The boolean is false when the selection is textual, allowing the
// manager to use ReadText without changing the clipboard.
func (c SystemClipboard) ReadImage() (string, bool, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "", false, nil
	}

	xclip, err := exec.LookPath("xclip")
	if err != nil {
		// xsel remains a valid text fallback, but its generic output mode does
		// not provide a safe MIME discovery path for image history.
		return "", false, nil
	}

	directory := c.imageDirectory
	if directory == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false, fmt.Errorf("obter diretorio pessoal para imagem do clipboard: %w", err)
		}
		directory = filepath.Join(home, "Imagens", "Captures")
	}

	return readImageWithXclip(xclip, directory, c.commandOutput())
}

func (SystemClipboard) WriteText(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("escrever clipboard: %w", err)
	}

	return nil
}

func (SystemClipboard) WriteImage(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("ler imagem para clipboard: %w", err)
	}

	if xclip, err := exec.LookPath("xclip"); err == nil {
		if err := exec.Command(xclip, imageClipboardArgs(path)...).Run(); err != nil {
			return fmt.Errorf("escrever imagem no clipboard com xclip: %w", err)
		}
		return nil
	}

	xsel, err := exec.LookPath("xsel")
	if err != nil {
		return errors.New("clipboard de imagem indisponivel: instale xclip ou xsel")
	}

	imageFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("abrir imagem para clipboard: %w", err)
	}
	defer imageFile.Close()

	command := exec.Command(xsel, "--clipboard", "--input", "--mime-type", imageTargetForPath(path))
	command.Stdin = imageFile
	if err := command.Run(); err != nil {
		return fmt.Errorf("escrever imagem no clipboard com xsel: %w", err)
	}
	return nil
}

func imageClipboardArgs(path string) []string {
	return []string{"-selection", "clipboard", "-target", imageTargetForPath(path), "-i", path}
}

type clipboardCommandOutput func(name string, args ...string) ([]byte, error)

func runClipboardCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (c SystemClipboard) commandOutput() clipboardCommandOutput {
	if c.output != nil {
		return c.output
	}
	return runClipboardCommand
}

func readTextWithXclip(command string, output clipboardCommandOutput) (string, error) {
	targets, err := output(command, xclipTargetsArgs()...)
	if err != nil {
		return "", fmt.Errorf("consultar tipos do clipboard: %w", err)
	}

	targetList := string(targets)
	// An image selection takes precedence over any auxiliary text target. This
	// prevents screenshot providers and clipboard managers from exposing an
	// image identifier/hash as a new text history row.
	if clipboardHasImageTarget(targetList) {
		return "", ErrClipboardNotText
	}

	target := textClipboardTarget(targetList)
	if target == "" {
		return "", ErrClipboardNotText
	}

	text, err := output(command, xclipTextArgs(target)...)
	if err != nil {
		return "", fmt.Errorf("ler alvo textual %q: %w", target, err)
	}
	return string(text), nil
}

func readImageWithXclip(command, directory string, output clipboardCommandOutput) (string, bool, error) {
	targets, err := output(command, xclipTargetsArgs()...)
	if err != nil {
		return "", false, fmt.Errorf("consultar tipos de imagem do clipboard: %w", err)
	}

	target := imageClipboardTarget(string(targets))
	if target == "" {
		return "", false, nil
	}

	data, err := output(command, xclipTextArgs(target)...)
	if err != nil {
		return "", false, fmt.Errorf("ler alvo de imagem %q: %w", target, err)
	}
	path, err := persistClipboardImage(directory, target, data)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

func persistClipboardImage(directory, target string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("imagem vazia no clipboard")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("criar diretorio de imagens do clipboard: %w", err)
	}

	digest := sha256.Sum256(data)
	path := filepath.Join(directory, fmt.Sprintf("clipboard-%x%s", digest[:], imageFileExtension(target)))
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("verificar imagem do clipboard: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("salvar imagem do clipboard: %w", err)
	}
	return path, nil
}

func readTextWithXsel(command string, output clipboardCommandOutput) (string, error) {
	text, err := output(command, xselTextArgs()...)
	if err != nil {
		return "", fmt.Errorf("ler alvo textual do xsel: %w", err)
	}
	return string(text), nil
}

func xclipTargetsArgs() []string {
	return []string{"-selection", "clipboard", "-out", "-target", "TARGETS"}
}

func xclipTextArgs(target string) []string {
	return []string{"-selection", "clipboard", "-out", "-target", target}
}

func xselTextArgs() []string {
	return []string{"--clipboard", "--output", "--mime-type", "text/plain"}
}

func clipboardHasImageTarget(targets string) bool {
	return imageClipboardTarget(targets) != ""
}

func imageClipboardTarget(targets string) string {
	preferred := []string{"image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif", "image/bmp"}
	available := make(map[string]string)
	for _, target := range strings.Fields(targets) {
		normalized := strings.ToLower(strings.TrimSpace(target))
		if strings.HasPrefix(normalized, "image/") {
			available[normalized] = strings.TrimSpace(target)
		}
	}
	for _, target := range preferred {
		if value, ok := available[target]; ok {
			return value
		}
	}
	for _, target := range available {
		return target
	}
	return ""
}

func imageTargetForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/png"
	}
}

func imageFileExtension(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	default:
		return ".png"
	}
}

func textClipboardTarget(targets string) string {
	preferred := []string{
		"UTF8_STRING",
		"text/plain;charset=utf-8",
		"text/plain",
		"TEXT",
		"STRING",
	}
	available := make(map[string]string)
	for _, target := range strings.Fields(targets) {
		normalized := strings.ToLower(strings.TrimSpace(target))
		if normalized != "" {
			available[normalized] = strings.TrimSpace(target)
		}
	}
	for _, target := range preferred {
		if value, ok := available[strings.ToLower(target)]; ok {
			return value
		}
	}
	return ""
}
