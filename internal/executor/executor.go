package executor

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type ScriptType int

const (
	TypeScript ScriptType = iota
	TypeCompilable
)

func (t ScriptType) String() string {
	switch t {
	case TypeScript:
		return "script"
	case TypeCompilable:
		return "compilable"
	default:
		return "unknown"
	}
}

type Script struct {
	Name string
	Path string
	Type ScriptType
}

func FindScripts(dir string) ([]Script, error) {
	var scripts []Script

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if name != "." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if name == "scripty" || name == "scripty.exe" {
			return nil
		}

		ext := filepath.Ext(name)
		relPath, _ := filepath.Rel(dir, path)

		switch ext {
		case ".sh", ".bash", ".zsh", ".fish",
			".py", ".pyw",
			".js", ".mjs", ".cjs",
			".rb",
			".pl", ".pm",
			".lua",
			".ts", ".tsx":
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeScript})
			return nil
		case ".go":
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeCompilable})
			return nil
		case ".c", ".cc", ".cpp", ".cxx":
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeCompilable})
			return nil
		case ".rs":
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeCompilable})
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&0o111 != 0 {
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeScript})
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		var shebang [2]byte
		f.Read(shebang[:])
		f.Close()
		if shebang[0] == '#' && shebang[1] == '!' {
			scripts = append(scripts, Script{Name: relPath, Path: path, Type: TypeScript})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})
	return scripts, nil
}

func Command(s Script) (*exec.Cmd, error) {
	ext := filepath.Ext(s.Name)
	switch ext {
	case ".sh", ".bash":
		return exec.Command("bash", s.Path), nil
	case ".zsh":
		return exec.Command("zsh", s.Path), nil
	case ".fish":
		return exec.Command("fish", s.Path), nil
	case ".py", ".pyw":
		return exec.Command("python3", s.Path), nil
	case ".js", ".mjs", ".cjs":
		return exec.Command("node", s.Path), nil
	case ".rb":
		return exec.Command("ruby", s.Path), nil
	case ".pl", ".pm":
		return exec.Command("perl", s.Path), nil
	case ".lua":
		return exec.Command("lua", s.Path), nil
	case ".ts", ".tsx":
		return exec.Command("npx", "tsx", s.Path), nil
	default:
		return exec.Command(s.Path), nil
	}
}

func BinaryPath(s Script) string {
	hash := md5.Sum([]byte(s.Path))
	return filepath.Join(os.TempDir(), fmt.Sprintf("scripty_%x", hash[:4]))
}

func CompileCommand(s Script) *exec.Cmd {
	out := BinaryPath(s)
	ext := filepath.Ext(s.Name)
	switch ext {
	case ".go":
		return exec.Command("go", "build", "-o", out, s.Path)
	case ".c":
		return exec.Command("gcc", "-o", out, s.Path)
	case ".cc", ".cpp", ".cxx":
		return exec.Command("g++", "-o", out, s.Path)
	case ".rs":
		return exec.Command("rustc", "-o", out, s.Path)
	default:
		return nil
	}
}

func RunCompiledCmd(s Script) *exec.Cmd {
	return exec.Command(BinaryPath(s))
}

func Cleanup(name string) {
	hash := md5.Sum([]byte(name))
	os.Remove(filepath.Join(os.TempDir(), fmt.Sprintf("scripty_%x", hash[:4])))
}
