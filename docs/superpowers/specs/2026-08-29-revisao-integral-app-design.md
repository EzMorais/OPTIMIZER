# Revisão integral do app — Design

## Objetivo

Tornar funcionais e previsíveis os fluxos existentes do Optimizer: inicialização, perfis de uso, ajustes, MTU, rede, diagnóstico, histórico, telemetria e restauração.

## Escopo aprovado

- Remover funções frontend duplicadas e corrigir IDs divergentes entre HTML e JavaScript.
- Garantir que cada operação tenha estado de carregamento, sucesso e erro, sempre restaurando os controles no `finally`.
- Evitar diagnósticos duplicados e chamadas concorrentes que deixam a UI presa.
- Corrigir atualização dos cards de perfil após aplicar/restaurar/verificar.
- Manter a política de segurança existente: simulação não grava, aplicação usa histórico e ponto de restauração quando habilitado.
- Cobrir regressões com testes frontend e Go existentes, adicionando testes apenas para comportamentos corrigidos.

## Fora do escopo

Login, cobrança, automação contínua, corretor de rotas e instalador assinado continuam fora desta revisão.

## Critérios de aceitação

1. Nenhuma aba fica permanentemente em carregamento após erro de binding ou operação do sistema.
2. Os botões não apontam para elementos inexistentes e não ficam desabilitados após falha.
3. O frontend tem uma única implementação de cada fluxo público, especialmente MTU.
4. Os testes automatizados passam; falhas exclusivamente ambientais são documentadas.
