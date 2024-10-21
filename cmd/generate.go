package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/weverton-souza/nexus-ops/parser" // Substitua pelo caminho correto do seu módulo
)

var (
	dir string
)

// init adiciona o comando generate e define suas flags
func init() {
	generateCmd.Flags().StringVarP(&dir, "directory", "d", ".", "Diretório raiz do projeto Java")
	rootCmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Gera arquivos JSON separados para cada classe nos arquivos Java",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Gerando arquivos JSON para as classes no diretório: %s\n", dir)

		// Chama a função ParseProject, que processa o diretório e salva os JSONs
		err := parser.ParseProject(dir)
		if err != nil {
			log.Fatalf("Erro ao parsear o projeto: %v", err)
		}

		fmt.Println("Arquivos JSON gerados com sucesso para cada classe.")
	},
}
