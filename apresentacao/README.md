# Optimizer

**Otimizador de PC para Windows que mede antes de mexer — e deixa desfazer tudo, sempre.**

App desktop em Go (.exe), vendido por assinatura semanal ou mensal, com versão gratuita generosa e recursos premium.

---

## O problema

"Otimizador de PC" virou sinônimo de golpe leve: o app assusta com *"encontramos 3.847 problemas"*, cobra para consertar o básico, e o usuário nunca sabe o que foi alterado nem como voltar atrás.

O concorrente mais forte hoje é gratuito e oficial — o Microsoft PC Manager. Mas ele não automatiza nada de forma contínua e trata computador de trabalho igual a computador pessoal.

**A aposta deste produto:** medir de verdade, explicar em português claro, mostrar o risco, e nunca cobrar pelo botão de desfazer.

---

## O que o app faz

### Diagnóstico que mostra números reais

Lê o estado atual da máquina e diz o que encontrou — com o valor medido, não com suposição. Se o PC está bem, o app diz que está bem. Nunca infla contagem de problemas.

### Catálogo com interruptor e explicação simples

Cada otimização é um liga/desliga com: o que faz, em português sem jargão; o risco; e a ressalva honesta quando o ganho é pequeno. Item que não vale a pena vem desmarcado — e explicando por quê.

### Desfazer sempre, e de graça

Ponto de restauração automático antes de qualquer alteração e histórico de tudo que foi mexido. Um clique volta ao estado anterior exato. Isso é rede de segurança básica, não recurso premium.

### Duas abas: Pessoal e Trabalho

O mesmo PC tem dois usos, e eles pedem coisas diferentes. No perfil Trabalho a barra de segurança é mais alta: nada que a TI da empresa possa depender é tocado. **Nenhum concorrente pesquisado — pago, grátis ou open-source — tem isso.**

### Teste de internet que mede de verdade

O app descobre o tamanho máximo de pacote que a sua conexão realmente entrega, mostra os comandos para você conferir na mão, e só sugere ajuste quando a medição justifica. Se está tudo certo, ele diz que não há nada a fazer.

### O que o app se recusa a fazer

Tem uma lista de "otimizações" famosas que o app **não** oferece — mitos, placebos e ajustes que trocam segurança por 2% de velocidade — cada uma com a explicação de por que ficou de fora. É argumento de venda, não omissão.

### Automação contínua *(premium)*

O app cuida do PC sozinho, em segundo plano, mantendo o perfil escolhido sem o usuário precisar abrir nada.

### Corretor de rotas de internet *(premium)*

Hub ao vivo de rota e latência, inspirado no que ExitLag e NoPing fazem — para jogos e chamadas.

---

## Grátis x Premium

| | Grátis | Premium |
|---|---|---|
| Diagnóstico completo | ✅ | ✅ |
| Todas as otimizações manuais | ✅ | ✅ |
| Ponto de restauração e desfazer | ✅ | ✅ |
| Teste de internet | ✅ | ✅ |
| Automação contínua | — | ✅ |
| Duas abas ao mesmo tempo | — | ✅ |
| Histórico completo | — | ✅ |
| Corretor de rotas | — | ✅ |

**Preço (hipótese a validar):** semanal R$ 7,90 · mensal R$ 19,90 · anual R$ 149,90 · empresarial R$ 14,90 por máquina/mês. Pagamento por PIX e cartão.

---

## Como está hoje

O **motor do produto já funciona** e foi testado numa máquina real: ele diagnostica, aplica, confirma que a alteração pegou, registra no histórico e desfaz. A medição de internet foi conferida contra o próprio Windows e bateu exatamente.

Falta a interface gráfica, o login e a cobrança.

Uma amostra do que ele já responde hoje está em [`demonstracao.md`](./demonstracao.md).

---

## Para saber mais

| | |
|---|---|
| Pesquisa de mercado e concorrentes | [`../docs/pesquisa-mercado.md`](../docs/pesquisa-mercado.md) |
| Decisões de produto | [`../docs/decisoes.md`](../docs/decisoes.md) |
| Catálogo técnico das otimizações | [`../docs/catalogo-windows.md`](../docs/catalogo-windows.md) |
| Arquitetura | [`../docs/arquitetura-app-desktop.md`](../docs/arquitetura-app-desktop.md) |
