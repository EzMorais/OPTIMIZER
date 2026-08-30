const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

function carregarControlador(elementos = {}, colecoes = {}) {
  const listeners = {};
  const document = {
    readyState: "loading",
    querySelector: (seletor) => elementos[seletor] || null,
    querySelectorAll: (seletor) => colecoes[seletor] || [],
    addEventListener: (evento, handler) => { listeners[evento] = handler; },
  };
  const contexto = {
    clearInterval,
    clearTimeout,
    console,
    document,
    setInterval,
    setTimeout,
    window: { document },
  };
  const codigo = fs.readFileSync(path.join(__dirname, "dist", "app.js"), "utf8");
  vm.createContext(contexto);
  vm.runInContext(codigo, contexto, { filename: "app.js" });
  return contexto;
}

test("abre Ajustes Avan\u00e7ados sem depender de uma vari\u00e1vel global inexistente", () => {
  const ui = carregarControlador();

  assert.equal(ui.deveDiagnosticarAoAbrirAjustes(null), true);
  assert.equal(ui.deveDiagnosticarAoAbrirAjustes({ itens: [] }), true);
  assert.equal(ui.deveDiagnosticarAoAbrirAjustes({ itens: [{ id: "visual.menu-show-delay" }] }), false);
});

test("carrega os perfis de rede no painel exibido pela interface", async () => {
  const painel = { innerHTML: "" };
  const ui = carregarControlador({ "#perfis-lista": painel });
  vm.runInContext("App = { ListarPerfisRede: async () => [] };", ui);

  await ui.carregarPerfis();

  assert.match(painel.innerHTML, /Nenhum perfil de rede disponível/);
});

test("mede a latencia usando o servidor de teste exibido na aba de rede", async () => {
  const elementos = {
    "#destino": { value: "1.1.1.1" },
    "#btn-medir-antes": { disabled: false, innerHTML: "" },
    "#btn-medir-depois": { disabled: true, innerHTML: "" },
    "#lat-antes-val": { textContent: "—" },
    "#comparativo-resultado": { innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = { MedirRedeAntes: async () => ({
      host: "1.1.1.1", minRTT: 10, avgRTT: 12, maxRTT: 15,
      stdDev: 1, packetsSent: 20, packetsLost: 0, lossPercent: 0
    }) };
    Visualizer.log = () => {};
  `, ui);

  await ui.medirAntes();

  assert.equal(elementos["#btn-medir-depois"].disabled, false);
  assert.equal(elementos["#lat-antes-val"].textContent, "12 ms");
  assert.match(elementos["#comparativo-resultado"].innerHTML, /Medição Base/);
});

test("conclui o diagnóstico mesmo sem os controles removidos de restauração e histórico", async () => {
  const elementos = {
    "#lista": { innerHTML: "" },
    "#resumo": { innerHTML: "" },
    "#visao-geral": { innerHTML: "" },
    "#admin-area": { innerHTML: "" },
    "#tab-badge-otim": { textContent: "0" },
    "#btn-desfazer-tudo": { disabled: false, innerHTML: "" },
    "#btn-rever": { addEventListener: () => {} },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      Diagnosticar: async () => ({
        admin: false, caminhoHistorico: "C:/temp/history.jsonl", total: 1,
        aplicados: 0, recomendadosPendentes: 1, pendentesDesfazer: 0, itens: []
      }),
      ResumoVisao: async () => { throw new Error("não deve executar um segundo diagnóstico"); }
    };
    Visualizer.log = () => {};
  `, ui);

  await assert.doesNotReject(() => ui.diagnosticar());
  assert.match(elementos["#resumo"].innerHTML, /1 otimização/);
});

test("atualiza o contador da barra de ações que existe na interface", () => {
  const elementos = {
    "#selecionados-count": { textContent: "0" },
    "#selecionados-sub": { textContent: "" },
    "#btn-aplicar": { disabled: true },
  };
  const ui = carregarControlador(elementos, {
    ".switch-control input:checked": [{ dataset: { id: "a" } }, { dataset: { id: "b" } }],
  });

  ui.atualizarContadorSelecionados();

  assert.equal(elementos["#selecionados-count"].textContent, 2);
  assert.equal(elementos["#btn-aplicar"].disabled, false);
});

test("marca apenas recomendações aplicáveis na sessão atual", () => {
  const elementos = {};
  const controles = [
    { dataset: { id: "usuario" }, disabled: false, checked: false },
    { dataset: { id: "maquina" }, disabled: false, checked: false },
  ];
  const ui = carregarControlador(elementos, {
    ".switch-control input[type=checkbox]": controles,
    ".switch-control input:checked": [],
  });
  vm.runInContext(`
    ultimoDiagnostico = { admin: false, itens: [
      { id: "usuario", recomendado: true, precisaAdmin: false },
      { id: "maquina", recomendado: true, precisaAdmin: true }
    ] };
    Visualizer.log = () => {};
    toast = () => {};
  `, ui);

  ui.marcarRecomendados();

  assert.equal(controles[0].checked, true);
  assert.equal(controles[1].checked, false);
});

test("simula ajustes sem exigir controles de restauração que não estão na tela", async () => {
  let pontoDeRestauracao = true;
  const ui = carregarControlador({}, {
    ".switch-control input[type=checkbox]:checked": [{ dataset: { id: "visual.menu-show-delay" } }],
    "#barra .dock-btn": [],
  });
  vm.runInContext(`
    App = { Aplicar: async (_ids, _simular, ponto) => { pontoDeRestauracao = ponto; return []; } };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    mostrarResultados = () => {};
    toast = () => {};
    carregarEstadoPerfis = async () => {};
    diagnosticar = async () => {};
  `, ui);
  ui.pontoDeRestauracao = pontoDeRestauracao;

  await ui.aplicar(true);

  assert.equal(ui.pontoDeRestauracao, false);
});

test("aplica perfil de uso sem abrir benchmark obrigatório", async () => {
  const ui = carregarControlador({ "#modal-corpo": { innerHTML: "" } });
  vm.runInContext(`
    listaPerfisUso = [{ key: "jogo", nome: "JOGO" }];
    let aplicacaoPerfil = 0;
    let benchmarkBase = 0;
    App = {
      AplicarPerfilUso: async () => { aplicacaoPerfil++; return [{ estado: "ok", nome: "Modo de jogo", mensagem: "Aplicado" }]; },
      IniciarBenchmarkBase: async () => { benchmarkBase++; return { sampleCount: 60 }; },
      ListarPerfisUso: async () => [],
      ObterPerfilAtivo: async () => ""
    };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    mostrarResultados = () => {};
    toast = () => {};
  `, ui);

  await ui.aplicarPerfilUso("jogo");

  assert.equal(vm.runInContext("aplicacaoPerfil", ui), 1);
  assert.equal(vm.runInContext("benchmarkBase", ui), 0);
});

test("não inicia dois diagnósticos concorrentes ao abrir ajustes", async () => {
  const ui = carregarControlador({
    "#lista": { innerHTML: "" },
    "#resumo": { innerHTML: "" },
    "#admin-area": { innerHTML: "" },
    "#tab-badge-otim": { textContent: "" },
    "#btn-desfazer-tudo": { disabled: false, innerHTML: "" },
    "#btn-rever": { addEventListener: () => {} },
  });
  vm.runInContext(`
    let diagnosticos = 0;
    App = { Diagnosticar: async () => {
      diagnosticos++;
      await new Promise((resolve) => setTimeout(resolve, 5));
      return { perfil: "pessoal", admin: false, total: 51, aplicados: 0, itens: [] };
    }};
    Visualizer.log = () => {};
    renderVisaoGeral = () => {};
    atualizarLista = () => {};
  `, ui);

  await Promise.all([ui.diagnosticar(), ui.diagnosticar()]);

  assert.equal(vm.runInContext("diagnosticos", ui), 1);
});

test("exibe item aplicado marcado e bloqueado", () => {
  const ui = carregarControlador();
  const html = vm.runInContext(`renderItem({
    id: "visual.menu-show-delay", nome: "Abrir menus na hora", descricao: "", detalhe: "",
    estado: "aplicado", recomendado: true, precisaAdmin: false, risco: "baixo"
  })`, ui);

  assert.match(html, /type="checkbox"[^>]*checked/);
  assert.match(html, /type="checkbox"[^>]*disabled/);
  assert.doesNotMatch(html, /Por que isso não vem marcado\?/);
});

test("verifica um perfil de uso com o formato real dos itens do diagnóstico", async () => {
  const elementos = { "#modal-corpo": { innerHTML: "" } };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    listaPerfisUso = [{
      key: "jogo", nome: "JOGO", tweakIds: ["jogos.game-mode"]
    }];
    App = { Diagnosticar: async () => ({ itens: [{
      id: "jogos.game-mode", nome: "Modo de Jogo", estado: "aplicado", detalhe: "Ativo"
    }] }) };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    abrirModal = () => {};
  `, ui);

  await ui.verificarPerfil("jogo");

  assert.match(elementos["#modal-corpo"].innerHTML, /Modo de Jogo/);
  assert.match(elementos["#modal-corpo"].innerHTML, /\[OK\]/);
});

test("mostra a medição atual de latência depois da comparação", async () => {
  const elementos = {
    "#destino": { value: "1.1.1.1" },
    "#btn-medir-depois": { disabled: false, innerHTML: "" },
    "#lat-depois-val": { textContent: "—" },
    "#comparativo-resultado": { innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    ultimoAntes = { host: "1.1.1.1", minRTT: 15, avgRTT: 18, maxRTT: 22, stdDev: 2, packetsSent: 20, packetsLost: 0, lossPercent: 0 };
    App = {
      MedirRedeDepois: async () => ({ host: "1.1.1.1", minRTT: 10, avgRTT: 12, maxRTT: 15, stdDev: 1, packetsSent: 20, packetsLost: 0, lossPercent: 0 }),
      RelatorioComparativo: async () => ({ interpretacao: "Melhorou", deltaLatencia: "-6 ms", deltaJitter: "-1 ms" })
    };
    Visualizer.log = () => {};
  `, ui);

  await ui.medirDepois();

  assert.equal(elementos["#lat-depois-val"].textContent, "12 ms");
  assert.match(elementos["#comparativo-resultado"].innerHTML, /Relatório de Impacto/i);
});

test("não mede o depois sem uma linha de base registrada", async () => {
  const elementos = {
    "#destino": { value: "1.1.1.1" },
    "#btn-medir-depois": { disabled: false, innerHTML: "" },
    "#comparativo-resultado": { innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    let chamadas = 0;
    App = { MedirRedeDepois: async () => { chamadas++; return {}; } };
    Visualizer.log = () => {};
  `, ui);

  await ui.medirDepois();

  assert.equal(vm.runInContext("chamadas", ui), 0);
  assert.match(elementos["#comparativo-resultado"].innerHTML, /medição base/i);
});

test("exibe o DNS em uso na interface ativa", async () => {
  const elementos = {
    "#dns-atual": { innerHTML: "" },
    "#btn-atualizar-dns": { addEventListener: () => {} },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = { ObterDNSAtual: async () => ({ interface: "Ethernet", servidores: ["192.168.1.1", "1.1.1.1"] }) };
  `, ui);

  await ui.carregarDNSAtual();

  assert.match(elementos["#dns-atual"].innerHTML, /Ethernet/);
  assert.match(elementos["#dns-atual"].innerHTML, /192\.168\.1\.1/);
});

test("aplica rapidamente os IPs do DNS escolhido no benchmark", async () => {
  const botao = { disabled: false, textContent: "Usar este DNS" };
  const ui = carregarControlador();
  vm.runInContext(`
    let ipsAplicados = [];
    App = { AplicarDNS: async (ips) => { ipsAplicados = ips; return { ok: true, mensagem: "DNS aplicado" }; } };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    toast = () => {};
  `, ui);

  await ui.usarDNS(["1.1.1.1", "1.0.0.1"], "Cloudflare", botao);

  assert.deepEqual(vm.runInContext("ipsAplicados", ui), ["1.1.1.1", "1.0.0.1"]);
  assert.equal(botao.textContent, "Usar este DNS");
  assert.equal(botao.disabled, false);
});

test("testa e renderiza a matriz de latencia de servidores de jogos", async () => {
  const elementos = {
    "#matriz-jogos-container": { innerHTML: "" },
    "#btn-testar-matriz-jogos": { disabled: false },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      MatrizPingJogos: async () => [
        { codigo: "br-sp", nome: "Brasil (São Paulo)", host: "189.38.95.95", localizacao: "BR", bandeira: "🇧🇷", pingMs: 14, jitterMs: 1.2, status: "otimo" },
        { codigo: "us-east", nome: "EUA Leste", host: "8.8.8.8", localizacao: "US", bandeira: "🇺🇸", pingMs: 118, jitterMs: 2.5, status: "regular" }
      ]
    };
  `, ui);

  await ui.testarMatrizJogos();

  assert.match(elementos["#matriz-jogos-container"].innerHTML, /Brasil \(São Paulo\)/);
  assert.match(elementos["#matriz-jogos-container"].innerHTML, /14 ms/);
  assert.match(elementos["#matriz-jogos-container"].innerHTML, /EUA Leste/);
  assert.equal(elementos["#btn-testar-matriz-jogos"].disabled, false);
});

test("executa limpeza profunda e flush da pilha de rede", async () => {
  const elementos = {
    "#flush-resultado": { innerHTML: "" },
    "#btn-flush-rede": { disabled: false },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      FlushingRede: async () => ({
        ok: true,
        mensagem: "4 operações de rede executadas com sucesso.",
        etapas: ["Limpeza de Cache DNS: Concluído", "Reset Winsock: Concluído"],
        erros: []
      })
    };
    Visualizer.log = () => {};
    toast = () => {};
  `, ui);

  await ui.executarFlushRede();

  assert.match(elementos["#flush-resultado"].innerHTML, /4 operações de rede/);
  assert.match(elementos["#flush-resultado"].innerHTML, /Limpeza de Cache DNS/);
  assert.equal(elementos["#btn-flush-rede"].disabled, false);
});

test("executa o benchmark GRC DNS com barras visuais de velocidade", async () => {
  const elementos = {
    "#dns-resultado": { innerHTML: "" },
    "#btn-testar-dns": { disabled: false },
  };
  const ui = carregarControlador(elementos, { ".btn-usar-dns": [] });
  vm.runInContext(`
    App = {
      BenchmarkDNS: async () => [
        { nome: "Cloudflare (1.1.1.1)", ips: ["1.1.1.1", "1.0.0.1"], avgRttMs: 11, privacidade: "Sem Logs", perda: 0, recomendado: true },
        { nome: "Google Public DNS", ips: ["8.8.8.8", "8.8.4.4"], avgRttMs: 15, privacidade: "Global", perda: 0, recomendado: false }
      ]
    };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    toast = () => {};
  `, ui);

  await ui.testarDNS();

  assert.match(elementos["#dns-resultado"].innerHTML, /Cloudflare/);
  assert.match(elementos["#dns-resultado"].innerHTML, /11 ms/);
  assert.match(elementos["#dns-resultado"].innerHTML, /Google Public DNS/);
  assert.equal(elementos["#btn-testar-dns"].disabled, false);
});

test("escaneia e limpa dispositivos fantasmas", async () => {
  const elementos = {
    "#pnp-resultado": { innerHTML: "" },
    "#btn-escanear-pnp": { disabled: false, innerHTML: "" },
    "#btn-limpar-pnp": { disabled: false, style: { display: "none" }, innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      ListarDispositivosFantasmas: async () => [
        { InstanceId: "USB\\VID_0000", FriendlyName: "Antigo Mouse USB Desconectado", Class: "Mouse" }
      ],
      LimparDispositivosFantasmas: async () => ({
        removidos: 1, erros: [], mensagem: "1 dispositivo(s) fantasma(s) removido(s) com sucesso."
      })
    };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
    toast = () => {};
  `, ui);

  await ui.escanearDispositivosFantasmas();
  assert.match(elementos["#pnp-resultado"].innerHTML, /Antigo Mouse USB/);
  assert.equal(elementos["#btn-limpar-pnp"].style.display, "inline-flex");

  await ui.limparDispositivosFantasmas();
  assert.equal(elementos["#btn-limpar-pnp"].disabled, false);
});

test("executa auditoria de integridade do sistema DISM e SFC", async () => {
  const elementos = {
    "#reparo-resultado": { innerHTML: "" },
    "#btn-auditar-reparo": { disabled: false, innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      AuditarReparo: async () => ({
        dismHealthy: true,
        sfcHealthy: true,
        interpretation: "Nenhuma violação de integridade ou corrupção encontrada nos arquivos de sistema.",
        dismOutput: "No component store corruption detected.",
        sfcOutput: "Windows Resource Protection did not find any integrity violations."
      })
    };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
  `, ui);

  await ui.auditarReparo();
  assert.match(elementos["#reparo-resultado"].innerHTML, /Sistema Saudável/);
  assert.match(elementos["#reparo-resultado"].innerHTML, /DISM CheckHealth/);
  assert.equal(elementos["#btn-auditar-reparo"].disabled, false);
});

test("mede e sintoniza MTU sem fragmentacao de pacotes", async () => {
  const elementos = {
    "#mtu-resultado": { innerHTML: "" },
    "#btn-medir-mtu": { disabled: false },
    "#destino": { value: "8.8.8.8" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = {
      MedirMTU: async () => ({
        mtuAtual: 1500,
        mtuCaminho: 1500,
        adaptador: "Ethernet",
        tipoAdaptador: "Ethernet",
        resumo: "MTU adequado",
        explicacao: "Pacotes transitam sem fragmentação.",
        podeAplicar: false,
        tentativas: []
      })
    };
    Visualizer.show = () => {};
    Visualizer.log = () => {};
  `, ui);

  await ui.medirMTU();
  assert.match(elementos["#mtu-resultado"].innerHTML, /MTU adequado/);
  assert.match(elementos["#mtu-resultado"].innerHTML, /MTU medido no caminho: 1500/);
  assert.equal(elementos["#btn-medir-mtu"].disabled, false);
});

test("mantem uma unica implementacao do fluxo de medicao de MTU", () => {
  const codigo = fs.readFileSync(path.join(__dirname, "dist", "app.js"), "utf8");
  assert.equal((codigo.match(/async function medirMTU\(/g) || []).length, 1);
});

test("mostra erro quando o diagnóstico do sistema falha em vez de manter o spinner", async () => {
  const elementos = {
    "#lista": { innerHTML: "" },
    "#resumo": { innerHTML: "" },
    "#visao-geral": { innerHTML: "" },
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`
    App = { Diagnosticar: async () => { throw new Error("Registro indisponível"); } };
    Visualizer.log = () => {};
  `, ui);

  await assert.doesNotReject(() => ui.diagnosticar());
  assert.match(elementos["#visao-geral"].innerHTML, /indisponível/);
  assert.match(elementos["#resumo"].innerHTML, /não foi possível concluir/i);
});

test("mantem uma unica implementacao dos fluxos publicos de DNS", () => {
  const codigo = fs.readFileSync(path.join(__dirname, "dist", "app.js"), "utf8");
  for (const nome of ["carregarDNSAtual", "testarDNS", "usarDNS"]) {
    assert.equal((codigo.match(new RegExp(`function ${nome}\\(`, "g")) || []).length, 1, nome);
  }
});

test("mantem os containers de Startup e Discos alinhados com o controlador", () => {
  const html = fs.readFileSync(path.join(__dirname, "dist", "index.html"), "utf8");
  const codigo = fs.readFileSync(path.join(__dirname, "dist", "app.js"), "utf8");

  for (const id of ["lista-startup", "lista-drives"]) {
    assert.match(html, new RegExp(`id=[\\"']${id}[\\"']`), `${id} ausente no HTML`);
    assert.match(codigo, new RegExp(`\\$\\(\\\"#${id}\\\"\\)`), `${id} ausente no controlador`);
  }
  assert.match(codigo, /carregarStartup\(\)/, "Startup não entra no preload");
  assert.match(codigo, /carregarDiscos\(\)/, "Discos não entra no preload");
});

test("não rejeita quando Startup, Discos ou MTU não têm controles na tela", async () => {
  const ui = carregarControlador();
  vm.runInContext("App = { ListarInicializacao: async () => [], ListarDiscos: async () => [] };", ui);
  await assert.doesNotReject(() => ui.carregarStartup());
  await assert.doesNotReject(() => ui.carregarDiscos());
  await assert.doesNotReject(() => ui.aplicarMTU());
});

test("mostra falha ao carregar perfis de rede sem rejeitar o evento da aba", async () => {
  const elementos = { "#perfis-lista": { innerHTML: "" } };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { ListarPerfisRede: async () => { throw new Error("rede indisponível"); } };`, ui);

  await assert.doesNotReject(() => ui.carregarPerfis());
  assert.match(elementos["#perfis-lista"].innerHTML, /rede indisponível/);
});

test("sinaliza telemetria indisponível quando o binding falha", async () => {
  const elementos = { "#telemetria-live-badge": { textContent: "SISTEMA ATIVO", style: {} } };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { ObterTelemetriaAoVivo: async () => { throw new Error("coletor indisponível"); } };`, ui);

  await assert.doesNotReject(() => ui.atualizarTelemetriaAoVivo());
  assert.equal(elementos["#telemetria-live-badge"].textContent, "TELEMETRIA INDISPONÍVEL");
});

test("sinaliza telemetria indisponível quando o backend retorna erro estruturado", async () => {
  const elementos = { "#telemetria-live-badge": { textContent: "SISTEMA ATIVO", style: {} } };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { ObterTelemetriaAoVivo: async () => ({ erro: "PowerShell indisponível" }) };`, ui);

  await assert.doesNotReject(() => ui.atualizarTelemetriaAoVivo());
  assert.equal(elementos["#telemetria-live-badge"].textContent, "TELEMETRIA INDISPONÍVEL");
});

test("carrega e renderiza resolução do timer do kernel", async () => {
  const elementos = {
    "#timerres-container": { innerHTML: "" },
    "#btn-timerres-05": { addEventListener: () => {} },
    "#btn-timerres-default": { addEventListener: () => {} },
    "#btn-testar-sleep-precision": { addEventListener: () => {} }
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { ObterTimerResolution: async () => ({ currentResolutionMs: 0.5, minResolutionMs: 15.625, maxResolutionMs: 0.5, isHighPrecision: true }) };`, ui);

  await assert.doesNotReject(() => ui.carregarTimerResolution());
  assert.match(elementos["#timerres-container"].innerHTML, /0.500 ms/);
  assert.match(elementos["#timerres-container"].innerHTML, /Alta Precisão/);
});

test("executa benchmark de sleep e exibe cálculo de jitter", async () => {
  const elementos = {
    "#btn-testar-sleep-precision": { disabled: false, innerHTML: "" },
    "#sleep-badge-score": { textContent: "" },
    "#sleep-precision-resultado": { innerHTML: "" }
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { MedirSleepPrecision: async () => ({ targetMs: 1.0, averageMs: 1.002, minMs: 0.998, maxMs: 1.005, stdDevMs: 0.002, jitterScore: "Excelente (Ultra Baixo Jitter)", samples: [1.001, 1.002] }) };`, ui);

  await assert.doesNotReject(() => ui.testarPrecisaoSleep());
  assert.equal(elementos["#sleep-badge-score"].textContent, "Excelente (Ultra Baixo Jitter)");
  assert.match(elementos["#sleep-precision-resultado"].innerHTML, /0.002 ms/);
});

test("escaneia e lista adaptadores PCIe com status MSI", async () => {
  const elementos = {
    "#btn-escanear-msi": { disabled: false, innerHTML: "" },
    "#msi-resultado": { innerHTML: "" }
  };
  const ui = carregarControlador(elementos);
  vm.runInContext(`App = { ListarDispositivosMSI: async () => [
    { id: "PCI\\\\VEN_10DE", nome: "NVIDIA GeForce RTX 4080", classe: "Display", caminhoRegistro: "HKLM\\\\PCI", msiSupported: true, statusRotulo: "MSI Mode Ativo" }
  ] };`, ui);

  await assert.doesNotReject(() => ui.escanearDispositivosMSI());
  assert.match(elementos["#msi-resultado"].innerHTML, /NVIDIA GeForce RTX 4080/);
  assert.match(elementos["#msi-resultado"].innerHTML, /MSI Mode Ativo/);
});

