# Demonstração

Saídas reais do motor rodando numa máquina Windows 11, hoje — antes de existir interface gráfica. É o mesmo motor que a interface vai usar.

## 1. Diagnóstico — mostra o número que mediu

```
ESTADO  NOME                                            DETALHE
[ ]     Abrir menus na hora                             Os menus ainda esperam um tempo antes de abrir.
                                                        Medimos agora: MenuShowDelay = 100 (alvo 10).
[ ]     Desligar a gravação de jogos em segundo plano   A gravação em segundo plano está ligada.
                                                        Medimos agora: GameDVR_Enabled = 1 (alvo 0).
[x]     Efeitos de transparência                        A transparência está desligada.
[x]     Mouse sem aceleração (movimento 1:1)            O mouse está no modo 1:1, sem aceleração.

4 de 9 itens já estão como você quer.
2 otimização(ões) recomendada(s) ainda não aplicada(s).
```

Repare: **4 de 9 já estavam certos, e o app diz isso** em vez de inventar problema. Só 2 itens são recomendados — o resto fica disponível, mas desmarcado e com a ressalva explicada.

## 2. Teste de internet — e o que ele se recusa a vender

```
O adaptador "Ethernet" está em MTU 1500 e o caminho até 8.8.8.8 aceita 1480 —
mas a rede avisa corretamente quando o pacote é grande demais, e o Windows já
se ajusta sozinho.

Confira você mesmo no Prompt de Comando:
  ping -n 1 -f -l 1452 8.8.8.8   → responde normalmente
  ping -n 1 -f -l 1453 8.8.8.8   → "Pacote precisa ser fragmentado, mas o
                                    sinalizador DF está definido."

Ajuste sugerido: MTU de "Ethernet" → 1480
Este ajuste é OPCIONAL: a medição mostrou que o Windows já está se adaptando
sozinho neste caminho.
```

O app **achou uma diferença real e mesmo assim disse que o ajuste é opcional**, porque medir mostrou que o Windows já estava resolvendo aquilo sozinho. É a diferença entre medir e vender urgência — e o usuário pode conferir o resultado na mão, com os dois comandos.

## 3. Aplicar e desfazer

```
$ aplicar visual.menu-show-delay
  ok   Abrir menus na hora — Os menus já abrem na hora.

$ desfazer --tudo
  ok   Abrir menus na hora — Voltou ao estado anterior.
```

O valor voltou exatamente ao que era antes (100), não a um "padrão de fábrica" chutado. Cada passo fica registrado num histórico que o usuário pode abrir e ler.
