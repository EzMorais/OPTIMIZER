# Guia de Perfis de Rede — OPTIMIZER

## 🎯 Quando Usar Cada Perfil

### 1. **Rede Rápida** (`network-fast`)
**Para:** Trabalho com transferências de arquivo, downloads grandes, streaming, upload intenso.

```
┌─────────────────────────────────────────────────────┐
│ REDE RÁPIDA — Throughput Máximo                    │
├─────────────────────────────────────────────────────┤
│ ✓ Desativa economia de energia do adaptador       │
│ ✓ Habilita RSS (distribuir RX entre cores)        │
│ ✓ Habilita RSC (agrupar pacotes)                  │
│ ✓ Reduz TIME_WAIT (30s em vez de 240s)            │
│ ✓ Aumenta portas efêmeras (até 65534)             │
├─────────────────────────────────────────────────────┤
│ ⚠️  Aumenta consumo de energia do adaptador        │
│ ⚠️  Considerar em notebooks apenas se plugado      │
└─────────────────────────────────────────────────────┘

EXEMPLO DE USO:
  optimizerctl perfil aplicar network-fast
  (depois: medir latência/throughput para confirmar melhoria)
```

**Esperado:**
- ⬇️ Latência: -5% a -15% (throughput pesado favorece bulk transfers)
- ⬆️ Throughput: +5% a +20% (se limitado por CPU/driver)
- ⬆️ CPU Usage: +2% a +5% (menos poder-saving overhead)

---

### 2. **Trabalho Remoto** (`network-remote`)
**Para:** RDP, SSH, Zoom, Teams, qualquer app que exige latência baixa e responsividade.

```
┌─────────────────────────────────────────────────────┐
│ TRABALHO REMOTO — Latência Baixa                   │
├─────────────────────────────────────────────────────┤
│ ✓ Desabilita Nagle's Algorithm (TCP_NODELAY)      │
│ ✓ Agresivo ACK frequency (aceitar overhead)        │
│ ✓ Não toca em poder (deixa economia se bateria)    │
├─────────────────────────────────────────────────────┤
│ ⚠️  Aumenta ACKs em rede de alta latência/perda     │
│ ⚠️  Efeito real maior em RDP/SSH; jogos modernos   │
│     usam UDP e não são afetados                    │
└─────────────────────────────────────────────────────┘

EXEMPLO DE USO:
  optimizerctl perfil aplicar network-remote
  (manter ativo durante jornada de trabalho remoto)
```

**Esperado:**
- ⬇️ Latência: -10% a -30% em RDP/SSH (reduz round-trip de ACK)
- ➡️ Jitter: Mais estável (menos espera de Nagle)
- ➡️ Throughput: Sem mudança perceptível (Nagle afeta responsividade, não throughput bulk)

---

### 3. **Desenvolvimento** (`network-dev`)
**Para:** Git clone/push, Docker pulls, API calls, compilação remota.

```
┌─────────────────────────────────────────────────────┐
│ DESENVOLVIMENTO — Sem Interferência P2P             │
├─────────────────────────────────────────────────────┤
│ ✓ Desativa Delivery Optimization (P2P updates)     │
│ ✓ Desativa energia adaptador (se importante)       │
│ ✓ Habilita RSS/RSC (benefício em muitas conexões) │
├─────────────────────────────────────────────────────┤
│ ⚠️  Pode reduzir autonomia em notebook              │
└─────────────────────────────────────────────────────┘

EXEMPLO DE USO:
  optimizerctl perfil aplicar network-dev
  (ideal durante day-time, desativar de noite)
```

**Esperado:**
- ⬆️ Throughput de Git/Docker: +5% (sem P2P em background)
- ⬇️ Latência: -2% a -5% (menos background bandwidth fight)
- ✓ Builds remotos mais estáveis (menos dropped connections)

---

### 4. **Apresentação** (`network-presentation`)
**Para:** Reuniões online, streaming, quando a conexão é crítica e deve ser estável.

```
┌─────────────────────────────────────────────────────┐
│ APRESENTAÇÃO — Estabilidade Máxima                 │
├─────────────────────────────────────────────────────┤
│ ✓ Desativa Delivery Optimization (sem P2P)         │
│ ✓ Remove limite de pacotes durante multimídia      │
│ ✓ Não toca em adaptador (deixa defaults estáveis) │
├─────────────────────────────────────────────────────┤
│ ⚠️  Manual: Selecionar Wi-Fi 5 GHz (menos congestion)│
│ ⚠️  Manual: Fechar downloads no fundo antes         │
└─────────────────────────────────────────────────────┘

EXEMPLO DE USO:
  optimizerctl perfil aplicar network-presentation
  (30 minutos antes da reunião; desfazer depois)
```

**Esperado:**
- ✓ Jitter: Reduzido (sem competição de P2P)
- ✓ Perda: 0% em rede de caixa
- ⬇️ Latência: Sem mudança (estabilidade, não ganho)

---

## 🛠️ Atalhos e Comandos Rápidos

### CLI (desenvolvimento/teste)

```bash
# Listar perfis disponíveis
optimizerctl perfil listar

# Aplicar um perfil (simulação — não grava nada)
optimizerctl perfil aplicar network-fast --simular

# Aplicar um perfil de verdade (requer admin)
optimizerctl perfil aplicar network-remote

# Medir latência antes de um ajuste
optimizerctl rede medir --destino 8.8.8.8

# Diagnóstico completo de rede
optimizerctl rede diag --destino 1.1.1.1

# Desfazer todos os ajustes de rede aplicados
optimizerctl desfazer --tudo
```

### Desktop GUI (quando pronto)

```
1. Abrir OPTIMIZER → aba "Internet e Rede"
2. Clicar "Medir Antes" (15 segundos)
3. Escolher perfil (ex: "Rede Rápida")
4. Revisar Caveats → "Aplicar Perfil"
5. Clicar "Medir Depois" (15 segundos)
6. Ver delta (antes 50ms → depois 47ms = -6%)
7. Se não gostar: "Desfazer"
```

---

## 📊 Fluxo Decisório: Qual Perfil?

```
┌─────────────────────────────────────────────┐
│ Qual é seu cenário agora?                  │
└─────────────────────────────────────────────┘
                    │
        ┌───────────┼───────────┐
        │           │           │
        ▼           ▼           ▼
   Trabalho    Download/    Reunião/
   Remoto      Upload      Apresentação
   (RDP/SSH)   Grande      (Zoom/Teams)
        │           │           │
        ▼           ▼           ▼
  "Trabalho   "Rede      "Apresentação"
   Remoto"    Rápida"
        │           │           │
        └───────────┼───────────┘
                    │
            Aplicar Perfil
             (medir antes)
                    │
                    ▼
         Latência melhorou 5%+?
         ┌─────────────┬──────────┐
         │             │          │
        SIM           NÃO       Piorou?
         │             │          │
         ▼             ▼          ▼
      Manter    Deixar     Desfazer
      Ativo     (overhead)  (rollback)
```

---

## ⚡ Dicas de Timing

| Cenário | Perfil | Quando | Quanto Tempo |
|---------|--------|--------|-------------|
| Desenvolvedor dia inteiro | Desenvolvimento | 9h-18h | Sempre ativo |
| Usuário doméstico | Rede Rápida | Downloads | Sob demanda |
| Executivo remoto | Trabalho Remoto | 8h-17h | Sempre ativo |
| Apresentador | Apresentação | 30 min antes | Temporário |
| Jogador online | Nenhum (UDP) | — | Não aplica |

---

## ✅ Checklist Antes e Depois

### Antes de Aplicar Perfil

- [ ] Internet estável (tester.com speedtest)
- [ ] Nenhum grande download em background
- [ ] Reinício de roteador/modem se possível
- [ ] Medição de latência baseline (20 pacotes)

### Depois de Aplicar Perfil

- [ ] Esperar 2-5 minutos (sistema se estabiliza)
- [ ] Medir latência novamente (mesmo host)
- [ ] Verificar se jitter diminuiu
- [ ] Testar aplicação real (RDP/Zoom/Git)
- [ ] Se ruim: desfazer com `optimizerctl desfazer --tudo`

---

## 📝 Notas Importantes

1. **Não é magia**: Ganhos são de 5%-20%, não 2x. Internet real é limitada por provedor/roteador, não por Windows.

2. **Reversibilidade garantida**: Cada perfil salva estado anterior. Desfazer = volta ao original.

3. **Honestidade acima de tudo**: Cada tweak inclui caveat (aviso) sobre trade-offs. Ex: "WiFi power sacing aumenta CPU".

4. **Medição real**: Sempre medir com `ping` real (20+ pacotes) antes/depois. Amostra única = ruído.

5. **Context matters**: Ganho em desenvolvimento (Git clone) ≠ ganho em gameplay (maioria joga UDP).

---

## 🔍 Quando Cada Tweak Importa

### TcpTimedWaitDelay (TIME_WAIT 240s → 30s)

**Importa se:**
- Seu app abre 5000+ conexões/hora (HTTP load testing, web scraping)
- Vê erro "Address already in use" regularmente
- Trabalha com microserviços locais (muitas conexões efêmeras)

**NÃO importa se:**
- Uso normal (browser, email, um programa por vez)

### Nagle's Algorithm (TCPNoDelay)

**Importa se:**
- Usa RDP, SSH, emuladores (telnet, etc)
- Quer click → feedback com <50ms latência
- Rejects todo overhead

**NÃO importa se:**
- Joga (UDP, não TCP)
- Baixa arquivo (throughput, não latência)

### RSS/RSC (offloads)

**Importa se:**
- Seu adaptador suporta (não é universal)
- CPU está 100% durante transferências gigabit
- Tem 4+ cores (benefício marginal em 2 cores)

**NÃO importa se:**
- Conexão < 300 Mbps real (CPU não é bottleneck)
- Notebook antigo (poder é limitado)

### Delivery Optimization (desativar P2P)

**Importa se:**
- Você tem internet com franquia (reduz upload)
- Está em 4G/5G
- Muitos PCs em casa (competição por banda)

**NÃO importa se:**
- Banda larga ilimitada folgada
- PC único em casa

---

## 📞 Suporte / Problemas

Se após aplicar um perfil você notar:
- Ping aumentado: reversível com `optimizerctl desfazer <tweak-id>`
- Roteador resetando: pode ser MTU (ajustar se necessário)
- VPN quebrando: possivelmente Winsock corrompido (reboot + desfazer)

Sempre há um `desfazer --tudo` para voltar ao original.
