package netdiag

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"optimizer/internal/console"
)

// ResultadoFlushRede resume a operação de limpeza e reset da pilha de rede.
type ResultadoFlushRede struct {
	Ok       bool     `json:"ok"`
	Mensagem string   `json:"mensagem"`
	Etapas   []string `json:"etapas"`
	Erros    []string `json:"erros"`
}

// ExecutarFlushRede limpa a cache DNS, renova tabelas de roteamento e reseta sockets.
func ExecutarFlushRede(ctx context.Context) ResultadoFlushRede {
	res := ResultadoFlushRede{
		Ok:     true,
		Etapas: []string{},
		Erros:  []string{},
	}

	comandos := []struct {
		nome string
		args []string
	}{
		{"Limpeza de Cache DNS (ipconfig /flushdns)", []string{"ipconfig", "/flushdns"}},
		{"Liberação de Concessão DHCP (ipconfig /release)", []string{"ipconfig", "/release"}},
		{"Renovação de Concessão DHCP (ipconfig /renew)", []string{"ipconfig", "/renew"}},
		{"Reset do Catálogo Winsock (netsh winsock reset)", []string{"netsh", "winsock", "reset"}},
	}

	for _, c := range comandos {
		cmd := exec.CommandContext(ctx, c.args[0], c.args[1:]...)
		console.HideWindow(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Alguns comandos podem precisar de permissões elevadas
			res.Erros = append(res.Erros, fmt.Sprintf("%s: %s", c.nome, strings.TrimSpace(string(out))))
		} else {
			res.Etapas = append(res.Etapas, fmt.Sprintf("%s: Concluído", c.nome))
		}
	}

	if len(res.Etapas) > 0 {
		res.Mensagem = fmt.Sprintf("%d operações de rede executadas com sucesso.", len(res.Etapas))
	} else if len(res.Erros) > 0 {
		res.Ok = false
		res.Mensagem = "Falha ao executar limpeza da pilha de rede."
	} else {
		res.Mensagem = "Limpeza de rede concluída."
	}

	return res
}
