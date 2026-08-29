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

  assert.equal(painel.innerHTML, "");
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
      ResumoVisao: async () => ({
        perfil: "pessoal", totalAjustes: 1, aplicados: 0,
        recomendadosPendentes: 1, pendentesDesfazer: 0, coberturaPercentual: 0, categorias: []
      })
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
  `, ui);
  ui.pontoDeRestauracao = pontoDeRestauracao;

  await ui.aplicar(true);

  assert.equal(ui.pontoDeRestauracao, false);
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
  assert.match(elementos["#comparativo-resultado"].innerHTML, /Relatório de Impacto Real/);
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

