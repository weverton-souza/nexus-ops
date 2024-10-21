package parser

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// Node representa um nó genérico da árvore de sintaxe
type Node struct {
	Type     string `json:"type"`
	Value    string `json:"value,omitempty"`
	Children []Node `json:"children,omitempty"`
}

// ParseProject percorre o diretório raiz, parseia arquivos Java e salva um arquivo JSON para cada classe, mantendo a estrutura de diretórios
func ParseProject(rootDir string) error {
	language := java.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(language)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Erro acessando %s: %v", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(d.Name()) != ".java" {
			return nil
		}

		// Extrai a árvore de sintaxe para cada arquivo Java
		tree, err := extractTree(path, parser)
		if err != nil {
			log.Printf("Erro parseando %s: %v", path, err)
			return nil
		}

		// Identifica o nome da classe e salva o arquivo JSON com a estrutura de diretórios
		err = saveClassToFileWithDir(tree, path, rootDir)
		if err != nil {
			log.Printf("Erro ao salvar arquivo JSON para %s: %v", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("erro ao percorrer diretórios: %v", err)
	}

	return nil
}

// extractTree parseia um único arquivo Java e retorna a estrutura genérica da árvore de sintaxe
func extractTree(filePath string, parser *sitter.Parser) (Node, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return Node{}, err
	}

	tree := parser.Parse(nil, source)
	root := tree.RootNode()

	genericTree := traverseTree(root, source)

	return genericTree, nil
}

// traverseTree percorre recursivamente a árvore de sintaxe e constrói a estrutura genérica com valores, exceto para certos tipos
func traverseTree(node *sitter.Node, source []byte) Node {
	n := Node{
		Type: node.Type(),
	}

	// Não atribuir valor para o nó do tipo "class_declaration"
	if node.Type() != "class_declaration" {
		// Extrai o texto do nó e remove espaços em branco e caracteres especiais
		rawValue := strings.TrimSpace(node.Content(source))
		n.Value = sanitizeValue(rawValue)
	}

	// Verifica se o nó contém filhos redundantes e, se sim, evita adicioná-los
	if n.Value != "" && isRedundantChildren(node, source) {
		return n
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil {
			n.Children = append(n.Children, traverseTree(child, source))
		}
	}

	return n
}

// isRedundantChildren verifica se os filhos de um nó são repetitivos e podem ser omitidos
func isRedundantChildren(node *sitter.Node, source []byte) bool {
	// Para nós como "scoped_identifier", se o valor for equivalente ao conteúdo de seus filhos, podemos omitir os children.
	if node.Type() == "scoped_identifier" {
		compositeValue := ""
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child != nil {
				compositeValue += strings.TrimSpace(child.Content(source)) + "."
			}
		}
		// Remover o último ponto extra
		compositeValue = strings.TrimSuffix(compositeValue, ".")
		// Se o valor do nó for equivalente ao valor composto dos filhos, é redundante
		return strings.TrimSpace(node.Content(source)) == compositeValue
	}
	return false
}

// sanitizeValue remove caracteres especiais como \r e \n do valor
func sanitizeValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

// saveClassToFileWithDir salva um arquivo JSON com o nome da classe encontrada no arquivo Java, replicando a estrutura de diretórios original
func saveClassToFileWithDir(tree Node, filePath string, rootDir string) error {
	className := ""
	for _, child := range tree.Children {
		if child.Type == "class_declaration" || child.Type == "interface_declaration" {
			for _, classChild := range child.Children {
				if classChild.Type == "identifier" {
					className = classChild.Value
					break
				}
			}
		}
	}

	if className == "" {
		log.Printf("Nenhuma classe ou interface encontrada em %s", filePath)
		return nil
	}

	outputBaseDir := "output"

	relativePath, err := filepath.Rel(rootDir, filepath.Dir(filePath))
	if err != nil {
		return fmt.Errorf("erro ao calcular o caminho relativo: %v", err)
	}

	outputDir := filepath.Join(outputBaseDir, relativePath)
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("erro ao criar diretórios: %v", err)
	}

	outputFileName := fmt.Sprintf("%s.json", className)
	outputFilePath := filepath.Join(outputDir, outputFileName)

	jsonData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao gerar JSON: %v", err)
	}

	err = ioutil.WriteFile(outputFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("erro ao escrever arquivo JSON: %v", err)
	}

	log.Printf("Classe %s salva em %s", className, outputFilePath)
	return nil
}
