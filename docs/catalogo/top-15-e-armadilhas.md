# Top 15 e armadilhas recusadas

> Síntese acionável do catálogo completo (ver [`../catalogo-windows.md`](../catalogo-windows.md)), cruzada por duas pesquisas independentes. As 15 otimizações abaixo são o coração do produto: melhor relação ganho-real/risco. As "armadilhas" são o texto pronto (PT-BR, simples) que a UI mostra quando o usuário pergunta por que um tweak famoso não está no app — é o diferencial de confiança contra CCleaner/IObit.

## Top 15 — ganho real, risco baixo

1. **Trocar plano de energia de "Economia de energia" para "Alto Desempenho"** (perfil trabalho/desktop) — maior impacto real e o mais frequentemente mal configurado de fábrica em PCs OEM. [`hardware-energia-gpu.md`](./hardware-energia-gpu.md#1-planos-de-energia--desempenho-máximo-oculto-ultimate-performance)
2. **Gerenciar programas de inicialização** — ganho de boot alto, mensurável, seguro, reversível a qualquer momento. [`servicos-inicializacao.md`](./servicos-inicializacao.md#parte-4--programas-de-inicialização)
3. **Corrigir gerenciamento de energia do adaptador de rede** ("permitir que o computador desligue este dispositivo") — resolve quedas/lentidão intermitente real, muito comum em notebook. [`rede.md`](./rede.md#8-adaptador-de-rede--energia)
4. **Desativar Xbox Game DVR em segundo plano** — ganho documentado em jogos, sem risco. [`registro.md`](./registro.md#xbox-game-bar--game-dvr)
5. **Restringir Delivery Optimization à rede local** — evita upload de atualizações saturando conexões limitadas/com cota. [`rede.md`](./rede.md#10-delivery-optimization)
6. **Garantir que a tarefa agendada de TRIM está ativa em SSD** — impacto de longo prazo alto, zero risco; já é automático, o app só precisa auditar. [`hardware-energia-gpu.md`](./hardware-energia-gpu.md#7-trim-e-otimizar-unidades)
7. **Desativar Energy Efficient Ethernet no adaptador cabeado** — reduz jitter/latência real (micro-stuttering) em jogos/VoIP. [`rede.md`](./rede.md#8-adaptador-de-rede--energia)
8. **Reduzir escopo de indexação do Windows Search** — mantém a busca rápida, corta indexação de HD externo/pastas irrelevantes, sem desativar o serviço inteiro. [`registro.md`](./registro.md#indexação-e-busca-windows-search)
9. **Ultimate Performance + Core Parking off** (só perfil desktop fixo) — ganho de consistência de clock em picos curtos. [`hardware-energia-gpu.md`](./hardware-energia-gpu.md#2-estacionamento-de-núcleos-core-parking)
10. **PCIe ASPM ajustado** quando há microtravamentos — resolve sintoma real de instabilidade, não só velocidade. [`hardware-energia-gpu.md`](./hardware-energia-gpu.md#3-pcie-link-state-power-management-aspm)
11. **`sfc /scannow` + `DISM /RestoreHealth`** como diagnóstico periódico de saúde — alto impacto quando há corrupção real de sistema. [`limpeza-manutencao.md`](./limpeza-manutencao.md#5-sfc-scannow-dism-restorehealth-chkdsk)
12. **DNS mais rápido (Cloudflare/Google), opcional** — pequeno ganho real na resolução de nomes, seguro, fácil de reverter. [`rede.md`](./rede.md#7-dns)
13. **HAGS testável com comparação antes/depois** — efeito variável por hardware, mas reversível e mensurável. [`hardware-energia-gpu.md`](./hardware-energia-gpu.md#5-hardware-accelerated-gpu-scheduling-hags)
14. **Remover/ocultar bloatware de forma gerenciada** — ganho de espaço/organização, comunicado honestamente, não vendido como velocidade. [`servicos-inicializacao.md`](./servicos-inicializacao.md#parte-5--bloatware-pré-instalado-24h225h2)
15. **`chkdsk /scan` + relatório SMART periódico** — pega o caso real "disco morrendo = PC lento", que tweak de registro nenhum resolve. [`limpeza-manutencao.md`](./limpeza-manutencao.md#5-sfc-scannow-dism-restorehealth-chkdsk), [`medicao-diagnostico.md`](./medicao-diagnostico.md#6-smart--saúde-do-disco)

## Armadilhas — o que o produto se recusa a fazer

Texto pronto para a UI (PT-BR, simples) quando o usuário perguntar "por que vocês não têm o tweak X que vi num vídeo":

| Tweak famoso | Texto que a UI mostra |
|---|---|
| Desativar mitigações de Spectre/Meltdown | *"Isso desliga proteções contra um tipo real de ataque que rouba dados de outros programas rodando no seu PC. O ganho de velocidade no dia a dia é pequeno (geralmente menos de 5%), mas o risco de segurança é real. Não vale a troca."* |
| "Desativar Nagle" para reduzir ping em jogos | *"A maioria dos jogos online modernos usa um tipo de conexão (UDP) que nem é afetado por essa configuração. É um mito antigo que continua sendo copiado de vídeo em vídeo."* |
| "Limitar Largura de Banda Reservável" do QoS (os "20% escondidos") | *"O Windows não guarda 20% da sua internet escondido. Esse número é só um teto máximo para programas raros que pedem prioridade de rede — na prática, você já usa 100% da sua internet."* |
| Desfragmentar SSD | *"SSD não tem peça girando, então 'fragmentação' não deixa ele mais lento. Só gasta a vida útil do disco à toa. O Windows já cuida disso sozinho de outro jeito (TRIM)."* |
| Limpeza/desfragmentação de registro | *"Não existe nenhum teste confiável mostrando que isso deixa o Windows mais rápido. Só existe risco: apagar a chave errada pode travar um programa ou o próprio Windows."* |
| Desativar completamente o arquivo de paginação (pagefile) | *"Parece que 'sobra' memória, mas o Windows usa esse arquivo também para se recuperar de travamentos. Desativar pode fazer programas fecharem sozinhos em vez do PC só ficar mais devagar."* |
| "Ativar" o disablelastaccess do NTFS | *"Essa dica já está desligada por padrão desde 2007. Ativá-la de novo não muda nada, porque já está feito."* |
| Desativar telemetria/DiagTrack vendido como "turbo de velocidade" | *"Isso é real para privacidade, mas não vai deixar seu PC nem 1% mais rápido. A gente não vende isso como otimização de desempenho porque seria te enganar."* |
| Excluir pastas genéricas do usuário (Downloads, Desktop, C:\ inteira) do antivírus | *"Esse é exatamente o truque que vírus e sequestradores de arquivo mais amam. A gente não oferece esse botão."* |
| `/ResetBase` automático no WinSxS | *"Isso apaga sua capacidade de desinstalar atualizações recentes se alguma der problema. Só oferecemos com aviso bem claro, nunca de forma automática."* |
| "Tweak de servidor" tipo LargeSystemCache no PC doméstico | *"Essa configuração é de 2003, para servidores de arquivo. No Windows 11 ela não faz nada."* |
| MTU manual "mágico" sem medir a rede | *"Mexer nisso sem testar direito quase sempre piora a internet, não melhora. Se você tem um problema real de MTU, a gente detecta e ajusta — não é um botão de 'acelerar'."* **Implementado** — o app mede o MTU real do caminho com eco ICMP "não fragmentar" (mesma técnica do `ping -f -l`), mostra os comandos para o usuário conferir na mão e só sugere ajuste quando a medição justifica. Ver [`rede.md`](./rede.md#como-o-produto-implementa-medição-nunca-chute--internalnetdiag). |
| "Liberar RAM" fechando processos do próprio Windows (explorer.exe, csrss.exe) | *"Fechar processos do próprio Windows para 'liberar memória' derruba o sistema. Isso não existe como otimização segura."* |
| Contador de "problemas encontrados" inflado | *"Se o seu PC está bem, a gente diz que está bem. Nunca vamos inventar um número assustador só para você clicar em 'corrigir'."* |

## Regra de ouro para o motor de recomendação

Um item só entra no perfil padrão (não-avançado) se, nas duas pesquisas independentes que embasaram este catálogo, o veredicto convergiu para `confirmado` ou `confirmado_com_ressalva` **e** o risco for `seguro`. Itens `moderado` entram no perfil avançado com aviso explícito. Itens `arriscado`/`perigoso` (mitigações de CPU, exclusões de antivírus em pastas genéricas, desativar pagefile) nunca entram em nenhum perfil sem confirmação adicional explícita do usuário.
