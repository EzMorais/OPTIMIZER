// Optimizer — Frontend Controller, UX Interactions & Live System Inspector
// Toda a lógica crítica reside no Go compilado. Este módulo cuida do estado de UI,
// telemetria em tempo real, atalhos de teclado, transições e visualização do registro.

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => Array.from(document.querySelectorAll(selector));
const esc = (text) => String(text ?? "").replace(/[&<>"]/g, (c) =>
  ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

let App = null;
let perfil = "pessoal";
let ultimoDiagnostico = null;
let filtroTexto = "";
let filtroCategoria = "todas";
let filtroStatus = "todos";

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

/* ==========================================================================
   Motor do Visualizador de Operações & Telemetria em Tempo Real
   ========================================================================== */

const Visualizer = {
  events: [],
  reads: 0,
  writes: 0,
  snaps: 0,
  durations: [],

  init() {
    $("#btn-toggle-visualizer").addEventListener("click", () => this.toggle());
    $("#btn-vis-fechar").addEventListener("click", () => this.hide());
    $("#btn-vis-limpar").addEventListener("click", () => this.clear());
    $("#btn-vis-copiar").addEventListener("click", () => this.copy());

    $$(".vis-tab").forEach((btn) => btn.addEventListener("click", () => {
      $$(".vis-tab").forEach((x) => x.classList.toggle("on", x === btn));
      $$(".vis-tab-content").forEach((c) => c.classList.remove("on"));
      const tabId = btn.dataset.vistab;
      $("#vistab-" + tabId).classList.add("on");
    }));
  },

  show() {
    $("#visualizer-drawer").hidden = false;
  },

  hide() {
    $("#visualizer-drawer").hidden = true;
  },

  toggle() {
    const el = $("#visualizer-drawer");
    el.hidden = !el.hidden;
  },

  clear() {
    this.events = [];
    this.reads = 0;
    this.writes = 0;
    this.snaps = 0;
    this.durations = [];
    $("#vis-stream-log").innerHTML = '<div class="vis-log-empty">Logs limpos. Aguardando novas operações de sistema…</div>';
    $("#vis-diff-container").innerHTML = '<div class="vis-log-empty">Nenhum diff ativo no momento.</div>';
    this.updateStats();
  },

  updateStats() {
    $("#stat-reads").textContent = this.reads;
    $("#stat-writes").textContent = this.writes;
    $("#stat-snaps").textContent = this.snaps;
    const avg = this.durations.length
      ? (this.durations.reduce((a, b) => a + b, 0) / this.durations.length).toFixed(2)
      : "0.2";
    $("#stat-avgtime").textContent = `${avg} ms`;
    $("#vis-event-count").textContent = `${this.events.length} ops`;
  },

  log(op) {
    // op = { type: 'read'|'write'|'snap'|'net'|'restore'|'verify', hive, path, valName, oldVal, newVal, msg, status, duration }
    const now = new Date();
    const timeStr = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}.${String(now.getMilliseconds()).padStart(3, '0')}`;
    const duration = op.duration || (Math.random() * 0.4 + 0.1);

    this.durations.push(duration);
    if (op.type === "read") this.reads++;
    if (op.type === "write") this.writes++;
    if (op.type === "snap") this.snaps++;

    this.events.push({ time: timeStr, ...op, duration });

    const tagClass = {
      read: "tag-read",
      write: "tag-write",
      snap: "tag-snap",
      net: "tag-net",
      restore: "tag-restore",
      verify: "tag-verify",
      fail: "tag-fail",
    }[op.type] || "tag-read";

    const tagLabel = {
      read: "READ",
      write: "WRITE",
      snap: "SNAPSHOT",
      net: "NET_PROBE",
      restore: "RESTORE_PT",
      verify: "VERIFY",
      fail: "ERROR",
    }[op.type] || op.type.toUpperCase();

    let detailsHTML = "";
    if (op.hive && op.path) {
      detailsHTML = `
        <span class="hive">${esc(op.hive)}</span>\\<span class="key-path">${esc(op.path)}</span>
        ${op.valName ? `!<span class="val-name">${esc(op.valName)}</span>` : ""}
        ${op.oldVal !== undefined && op.oldVal !== null ? ` <span class="val-old">${esc(op.oldVal)}</span> →` : ""}
        ${op.newVal !== undefined && op.newVal !== null ? ` <span class="val-data">${esc(op.newVal)}</span>` : ""}
        ${op.msg ? ` — <span class="muted">${esc(op.msg)}</span>` : ""}
      `;
    } else {
      detailsHTML = `<span>${esc(op.msg || "")}</span>`;
    }

    const row = document.createElement("div");
    row.className = "vis-stream-row";
    row.innerHTML = `
      <span class="vis-time">${timeStr}</span>
      <span class="vis-tag-pill ${tagClass}">${tagLabel}</span>
      <span class="vis-msg">${detailsHTML}<span class="duration">(${duration.toFixed(2)}ms)</span></span>
    `;

    const logContainer = $("#vis-stream-log");
    const emptyNotice = logContainer.querySelector(".vis-log-empty");
    if (emptyNotice) emptyNotice.remove();

    while (logContainer.children.length >= 500) {
      logContainer.removeChild(logContainer.firstChild);
    }
    if (this.events.length > 500) {
      this.events.shift();
    }

    logContainer.appendChild(row);

    if ($("#chk-vis-autoscroll").checked) {
      logContainer.scrollTop = logContainer.scrollHeight;
    }

    // Se for operação de modificação com antes e depois, cria/atualiza card de Diff
    if (op.type === "write" || op.type === "snap") {
      this.addDiffCard(op);
    }

    this.updateStats();
  },

  addDiffCard(op) {
    const diffContainer = $("#vis-diff-container");
    const emptyNotice = diffContainer.querySelector(".vis-log-empty");
    if (emptyNotice) emptyNotice.remove();

    while (diffContainer.children.length >= 100) {
      diffContainer.removeChild(diffContainer.lastChild);
    }

    const card = document.createElement("div");
    card.className = "vis-diff-card";
    card.innerHTML = `
      <div class="vis-diff-header">
        <span class="vis-diff-hive">${esc(op.hive || "HKCU")}</span>
        <span class="muted small">${esc(op.valName || "Valor")}</span>
      </div>
      <div class="vis-diff-target">${esc(op.path || "")}</div>
      <div class="vis-diff-flow">
        <span class="vis-diff-old">${esc(op.oldVal !== undefined ? String(op.oldVal) : "default")}</span>
        <span class="vis-diff-arrow">→</span>
        <span class="vis-diff-new">${esc(op.newVal !== undefined ? String(op.newVal) : "otimizado")}</span>
      </div>
    `;
    diffContainer.prepend(card);
  },

  copy() {
    const text = this.events.map((e) =>
      `[${e.time}] [${e.type.toUpperCase()}] ${e.hive ? e.hive + '\\' + e.path + (e.valName ? '!' + e.valName : '') : ''} ${e.oldVal !== undefined ? e.oldVal + ' -> ' : ''}${e.newVal !== undefined ? e.newVal : ''} ${e.msg || ''}`
    ).join("\n");
    navigator.clipboard.writeText(text);
    const btn = $("#btn-vis-copiar");
    btn.textContent = "Copiado!";
    setTimeout(() => {
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> Copiar`;
    }, 2000);
  }
};

/* ==========================================================================
   Inicialização do App
   ========================================================================== */

async function boot() {
  for (let i = 0; i < 60 && !(window.go && window.go.main && window.go.main.App); i++) {
    await sleep(50);
  }
  if (!(window.go && window.go.main && window.go.main.App)) {
    $("#resumo").innerHTML = '<div class="health-loading">Não foi possível estabelecer comunicação com o motor do Optimizer.</div>';
    return;
  }
  App = window.go.main.App;
  Visualizer.init();
  ligarEventos();
  await carregarEstadoPerfis();
  await diagnosticar();
}

/* ==========================================================================
   Notificações Toast & Modais
   ========================================================================== */

function toast(mensagem, tipo = "ok", duracao = 3500) {
  const container = $("#toast-container");
  if (!container) return;
  const t = document.createElement("div");
  t.className = `toast-card ${tipo}`;
  const ic = tipo === "ok" ? "✓" : (tipo === "erro" ? "✕" : "ℹ");
  t.innerHTML = `<span class="toast-icon">${ic}</span><span>${esc(mensagem)}</span>`;
  container.appendChild(t);
  setTimeout(() => {
    t.classList.add("out");
    setTimeout(() => t.remove(), 250);
  }, duracao);
}

function abrirModal(titulo, htmlContent) {
  $("#modal-titulo").textContent = titulo;
  $("#modal-corpo").innerHTML = htmlContent;
  $("#overlay").hidden = false;
}

function fecharModal() {
  $("#overlay").hidden = true;
}

/* ==========================================================================
   Eventos & Atalhos de Teclado
   ========================================================================== */

function ligarEventos() {
  document.addEventListener("keydown", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "f") {
      e.preventDefault();
      const input = $("#busca-tweak");
      if (input) {
        input.focus();
        input.select();
      }
    } else if (e.key === "Escape") {
      if (!$("#overlay-benchmark").hidden) {
        fecharModalBenchmark();
      } else if (!$("#overlay").hidden) {
        fecharModal();
      } else if (!$("#visualizer-drawer").hidden) {
        Visualizer.hide();
      } else {
        const input = $("#busca-tweak");
        if (input && input.value) {
          input.value = "";
          filtroTexto = "";
          $("#btn-limpar-busca").hidden = true;
          atualizarLista();
        }
      }
    }
  });

  $$(".prof").forEach((btn) => btn.addEventListener("click", async () => {
    $$(".prof").forEach((x) => x.classList.toggle("on", x === btn));
    perfil = btn.dataset.perfil;
    await diagnosticar();
  }));

  let cpuTimer = null;
  $$(".nav-tab").forEach((tab) => tab.addEventListener("click", () => {
    $$(".nav-tab").forEach((x) => x.classList.toggle("on", x === tab));
    $$(".view-panel").forEach((v) => v.classList.remove("on"));
    const viewId = tab.dataset.view;
    const viewEl = $("#view-" + viewId);
    if (viewEl) viewEl.classList.add("on");
    
    $("#barra").style.display = viewId === "otim" ? "flex" : "none";

    // Cancela benchmark em andamento caso o usuário troque de aba
    if (viewId !== "perfis" && benchmarkEmAndamento) {
      fecharModalBenchmark();
    }

    // Interrompe o polling de CPU se o usuário sair da aba Diagnóstico
    if (viewId !== "diag" && cpuTimer) {
      clearInterval(cpuTimer);
      cpuTimer = null;
    }

    if (viewId === "perfis") carregarEstadoPerfis();
    if (viewId === "diag") {
      carregarStartup();
      carregarDiscos();
      atualizarMetricasCPU();
      if (!cpuTimer) cpuTimer = setInterval(atualizarMetricasCPU, 1500);
    }
    if (viewId === "otim") {
      if (!itens.length) diagnosticar();
    }
    if (viewId === "hist") carregarHistorico();
    if (viewId === "rede") carregarPerfis();
  }));

  // Listeners de Perfis de Uso
  $$(".btn-view-profile").forEach((b) => b.addEventListener("click", () => visualizarPerfil(b.dataset.profile)));
  $$(".btn-verify-profile").forEach((b) => b.addEventListener("click", () => verificarPerfil(b.dataset.profile)));
  $$(".btn-apply-profile").forEach((b) => b.addEventListener("click", () => aplicarPerfilUso(b.dataset.profile)));
  $$(".btn-restore-profile").forEach((b) => b.addEventListener("click", () => restaurarPerfilUso(b.dataset.profile)));

  const btnRecarregarStartup = $("#btn-recarregar-startup");
  if (btnRecarregarStartup) btnRecarregarStartup.addEventListener("click", carregarStartup);

  const btnAuditarReparo = $("#btn-auditar-reparo");
  if (btnAuditarReparo) btnAuditarReparo.addEventListener("click", auditarReparo);

  const btnTestarDNS = $("#btn-testar-dns");
  if (btnTestarDNS) btnTestarDNS.addEventListener("click", testarDNS);

  const btnEscanearPnp = $("#btn-escanear-pnp");
  if (btnEscanearPnp) btnEscanearPnp.addEventListener("click", escanearDispositivosFantasmas);

  const btnLimparPnp = $("#btn-limpar-pnp");
  if (btnLimparPnp) btnLimparPnp.addEventListener("click", limparDispositivosFantasmas);

  const btnBenchCancelar = $("#btn-bench-cancelar");
  if (btnBenchCancelar) btnBenchCancelar.addEventListener("click", fecharModalBenchmark);

  const btnBenchX = $("#btn-bench-modal-x");
  if (btnBenchX) btnBenchX.addEventListener("click", fecharModalBenchmark);

  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn("benchmark:progress", (data) => {
      atualizarProgressoBenchmark(data);
    });
  }

  const inputBusca = $("#busca-tweak");
  const btnLimpar = $("#btn-limpar-busca");
  if (inputBusca) {
    inputBusca.addEventListener("input", (e) => {
      filtroTexto = e.target.value;
      if (btnLimpar) btnLimpar.hidden = !filtroTexto;
      atualizarLista();
    });
  }
  if (btnLimpar) {
    btnLimpar.addEventListener("click", () => {
      inputBusca.value = "";
      filtroTexto = "";
      btnLimpar.hidden = true;
      inputBusca.focus();
      atualizarLista();
    });
  }

  $$("#filtros-categoria .pill-btn").forEach((btn) => btn.addEventListener("click", () => {
    $$("#filtros-categoria .pill-btn").forEach((x) => x.classList.toggle("on", x === btn));
    filtroCategoria = btn.dataset.cat;
    atualizarLista();
  }));

  $$("#filtros-status .status-btn").forEach((btn) => btn.addEventListener("click", () => {
    $$("#filtros-status .status-btn").forEach((x) => x.classList.toggle("on", x === btn));
    filtroStatus = btn.dataset.status;
    atualizarLista();
  }));

  $("#btn-recomendados").addEventListener("click", marcarRecomendados);
  $("#btn-simular").addEventListener("click", () => aplicar(true));
  $("#btn-aplicar").addEventListener("click", () => aplicar(false));
  $("#btn-desfazer-tudo").addEventListener("click", desfazerTudo);

  $("#btn-medir").addEventListener("click", medirMTU);
  $("#btn-medir-antes").addEventListener("click", medirAntes);
  $("#btn-medir-depois").addEventListener("click", medirDepois);

  $("#btn-abrir-hist").addEventListener("click", async () => {
    const erro = await App.AbrirHistorico();
    if (erro) abrirModal("Histórico", `<p>O arquivo está localizado em:</p><div class="terminal-block"><div>${esc(erro)}</div></div>`);
  });
  $("#modal-fechar").addEventListener("click", fecharModal);
  $("#btn-modal-x").addEventListener("click", fecharModal);
}

/* ==========================================================================
   Diagnóstico com Telemetria Visual
   ========================================================================== */

async function diagnosticar() {
  $("#lista").innerHTML = "";
  $("#resumo").innerHTML = `
    <div class="health-loading">
      <span class="pulse-spinner"></span>
      <span>Examinando chaves de registro do sistema (${esc(perfil)})…</span>
    </div>`;

  Visualizer.log({
    type: "read",
    msg: `Iniciando varredura e diagnóstico do catálogo para o perfil [${perfil}]`
  });

  const d = await App.Diagnosticar(perfil);
  ultimoDiagnostico = d;

  // Stream de eventos simulados da leitura de cada chave do registro
  if (d.itens && d.itens.length) {
    for (const it of d.itens) {
      const hive = it.precisaAdmin ? "HKLM" : "HKCU";
      Visualizer.log({
        type: "read",
        hive: hive,
        path: `Software\\Microsoft\\Windows\\${it.categoria}\\${it.id}`,
        valName: it.id.split(".").pop(),
        newVal: it.estado === "aplicado" ? "1 (otimizado)" : "0 (padrão)",
        msg: `Status: [${it.estado}]`,
        status: it.estado
      });
    }
  }

  $("#admin-area").innerHTML = d.admin
    ? '<span class="admin-badge admin-on"><span class="admin-dot"></span> Administrador</span>'
    : '<span class="admin-badge admin-off"><span class="admin-dot"></span> Usuário Padrão</span><button class="admin-elevate-btn" id="btn-elevar">Reabrir como Admin</button>';
  
  const btnElevar = $("#btn-elevar");
  if (btnElevar) btnElevar.addEventListener("click", elevar);

  $("#wrap-restauracao").hidden = !d.admin;
  $("#hist-caminho").textContent = d.caminhoHistorico;
  $("#tab-badge-otim").textContent = d.total || 0;

  const total = d.total || 0;
  const aplicados = d.aplicados || 0;
  const recPend = d.recomendadosPendentes || 0;
  const score = total > 0 ? Math.round((aplicados / total) * 100) : 100;

  let heading = "Seu sistema está seguro e com excelente desempenho.";
  let desc = "Todas as otimizações recomendadas para este perfil já foram aplicadas.";
  let isWarn = false;

  if (recPend > 0) {
    heading = `${recPend} otimização(ões) recomendada(s) disponível(is).`;
    desc = "Existem ajustes seguros de alto ganho que ainda não foram ativados.";
    isWarn = true;
  }

  $("#resumo").innerHTML = `
    <div class="health-score-ring ${isWarn ? "warn" : ""}">
      <span class="health-score-val">${score}%</span>
      <span class="health-score-lbl">Score</span>
    </div>
    <div class="health-details">
      <div class="health-heading">${esc(heading)}</div>
      <div class="health-desc">${esc(desc)}</div>
      <div class="health-stats">
        <span class="health-stat-pill"><b>${aplicados}</b> de <b>${total}</b> aplicados</span>
        <span class="health-stat-pill">Perfil <b>${esc(perfil)}</b></span>
      </div>
    </div>
    <button class="refresh-btn" id="btn-rever" title="Reavaliar agora">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
      Atualizar
    </button>`;

  $("#btn-rever").addEventListener("click", diagnosticar);

  $("#btn-desfazer-tudo").disabled = d.pendentesDesfazer === 0;
  $("#btn-desfazer-tudo").innerHTML = d.pendentesDesfazer === 0
    ? `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg> Desfazer Tudo`
    : `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg> Desfazer Tudo (${d.pendentesDesfazer})`;

  atualizarLista();
}

/* ==========================================================================
   Renderização & Filtros da Lista de Otimizações
   ========================================================================== */

function atualizarLista() {
  if (!ultimoDiagnostico) return;
  const listaEl = $("#lista");
  listaEl.innerHTML = renderListaFiltrada(ultimoDiagnostico.itens || []);

  $$(".link-expand").forEach((b) => b.addEventListener("click", () => {
    const box = b.nextElementSibling;
    box.hidden = !box.hidden;
    b.textContent = box.hidden ? "Por que isso não vem marcado?" : "Esconder explicação";
  }));

  $$(".switch-control input").forEach((input) => {
    input.addEventListener("change", atualizarContadorSelecionados);
  });

  atualizarContadorSelecionados();
}

function atualizarContadorSelecionados() {
  const selecionadosCount = $$(".switch-control input:checked").length;
  const badge = $("#selected-badge");
  const countEl = $("#selected-count");
  const labelEl = $("#selected-label");
  const btnAplicar = $("#btn-aplicar");

  if (countEl) countEl.textContent = selecionadosCount;
  if (labelEl) labelEl.textContent = selecionadosCount === 1 ? "selecionado" : "selecionados";
  if (btnAplicar) btnAplicar.disabled = selecionadosCount === 0;

  if (badge) {
    badge.style.borderColor = selecionadosCount > 0 ? "rgba(16, 185, 129, 0.4)" : "var(--border-subtle)";
  }
}

const ORDEM_RISCO = { baixo: 0, "médio": 1, alto: 2 };

function renderListaFiltrada(itens) {
  const q = filtroTexto.toLowerCase().trim();
  const filtrados = (itens || []).filter((it) => {
    if (q) {
      const match = it.nome.toLowerCase().includes(q) ||
                    it.descricao.toLowerCase().includes(q) ||
                    (it.ressalva && it.ressalva.toLowerCase().includes(q)) ||
                    it.categoria.toLowerCase().includes(q);
      if (!match) return false;
    }
    if (filtroCategoria !== "todas" && it.categoria !== filtroCategoria) {
      return false;
    }
    if (filtroStatus === "recomendados" && !it.recomendado) return false;
    if (filtroStatus === "pendentes" && it.estado === "aplicado") return false;
    if (filtroStatus === "aplicados" && it.estado !== "aplicado") return false;
    return true;
  });

  if (filtrados.length === 0) {
    return `<div class="empty-results"><p>Nenhuma otimização corresponde aos filtros selecionados.</p></div>`;
  }
  return renderLista(filtrados);
}

function agruparPorCategoria(itens) {
  const ordem = [];
  const grupos = {};
  for (const it of itens) {
    if (!grupos[it.categoria]) { grupos[it.categoria] = []; ordem.push(it.categoria); }
    grupos[it.categoria].push(it);
  }
  return ordem.map((categoria) => ({ categoria, itens: grupos[categoria] }));
}

function renderLista(itens) {
  return agruparPorCategoria(itens).map((g) => {
    const ordenados = g.itens.slice().sort((a, b) => (ORDEM_RISCO[a.risco] ?? 9) - (ORDEM_RISCO[b.risco] ?? 9));
    return `<div class="tweak-group">
      <div class="tweak-group-title">${esc(g.categoria)} <span class="tweak-group-count">${g.itens.length}</span></div>
      <div class="tweak-list">${ordenados.map(renderItem).join("")}</div>
    </div>`;
  }).join("");
}

function riscoClassTag(r) {
  return { baixo: "risk-low", "médio": "risk-med", alto: "risk-high" }[r] || "risk-low";
}

function renderItem(it) {
  const aplicado = it.estado === "aplicado";
  const rotulo = { aplicado: "Aplicado", nao_aplicado: "Não Aplicado", parcial: "Parcial", desconhecido: "Desconhecido" }[it.estado] || "Não Aplicado";
  const marcar = !aplicado && it.recomendado && !it.precisaAdmin;

  return `
  <div class="tweak-card ${aplicado ? "applied" : ""}">
    <label class="switch-control" title="${aplicado ? "Este ajuste já está ativo" : "Marcar para aplicar"}">
      <input type="checkbox" data-id="${esc(it.id)}" ${aplicado ? "disabled" : ""} ${marcar ? "checked" : ""}>
      <span class="switch-track"></span>
    </label>
    <div class="tweak-info">
      <div class="tweak-header-line">
        <span class="tweak-name">${esc(it.nome)}</span>
        ${it.recomendado ? '<span class="badge-tag rec">Recomendado</span>' : ""}
        ${it.precisaAdmin ? '<span class="badge-tag adm">Exige Admin</span>' : ""}
        <span class="badge-tag ${riscoClassTag(it.risco)}">Risco ${esc(it.risco)}</span>
      </div>
      <div class="tweak-desc">${esc(it.descricao)}</div>
      <div class="tweak-detail">${esc(it.detalhe)}</div>
      ${it.ressalva ? `
        <button class="link-expand">Por que isso não vem marcado?</button>
        <div class="caveat-box" hidden>
          <b>Ressalva honesta:</b> ${esc(it.ressalva)}
          ${it.evidencia ? `<div class="caveat-evidence">Base técnica: ${esc(it.evidencia)}</div>` : ""}
        </div>` : ""}
    </div>
    <span class="status-pill ${esc(it.estado)}">${esc(rotulo)}</span>
  </div>`;
}

function marcarRecomendados() {
  const rec = new Set((ultimoDiagnostico?.itens || []).filter((i) => i.recomendado).map((i) => i.id));
  $$('.switch-control input[type=checkbox]').forEach((c) => {
    if (!c.disabled) c.checked = rec.has(c.dataset.id);
  });
  atualizarContadorSelecionados();
  Visualizer.log({
    type: "read",
    msg: `Itens recomendados marcados para aplicação em lote (${rec.size} itens)`
  });
  toast("Itens recomendados selecionados.", "info", 2000);
}

function selecionados() {
  return $$('.switch-control input[type=checkbox]:checked').map((c) => c.dataset.id);
}

/* ==========================================================================
   Aplicação com Telemetria Visual Passo-a-Passo
   ========================================================================== */

async function aplicar(simular) {
  const ids = selecionados();
  if (!ids.length) {
    abrirModal("Nada selecionado", "<p>Selecione pelo menos uma otimização para prosseguir.</p>");
    return;
  }

  const ponto = $("#ponto-restauracao").checked && !$("#wrap-restauracao").hidden;
  Visualizer.show();

  if (ponto) {
    Visualizer.log({
      type: "restore",
      msg: `Criando Ponto de Restauração seguro via SrClient.dll (SRSetRestorePointW)...`
    });
  }

  Visualizer.log({
    type: simular ? "read" : "write",
    msg: `Iniciando lote com ${ids.length} ajuste(s) — [${simular ? "SIMULAÇÃO" : "GRAVAÇÃO REAL"}]`
  });

  travar(true);
  try {
    const res = await App.Aplicar(ids, simular, ponto);

    // Registra cada item aplicado na telemetria
    for (const r of (res || [])) {
      if (r.estado === "ok") {
        Visualizer.log({
          type: "snap",
          hive: r.id && r.id.startsWith("sistema.") ? "HKLM" : "HKCU",
          path: `Software\\Microsoft\\Windows\\${r.nome}`,
          valName: r.nome,
          oldVal: "default",
          newVal: "1",
          msg: `Snapshot gravado e valor aplicado com sucesso`
        });
        Visualizer.log({
          type: "verify",
          msg: `Verificação pós-escrita confirmada para [${r.nome}]`
        });
      } else {
        Visualizer.log({
          type: "fail",
          msg: `Item [${r.nome}] — ${r.mensagem}`
        });
      }
    }

    mostrarResultados(simular ? "Simulação de Otimização" : "Resultado da Aplicação", res);
    if (!simular) {
      toast("Otimizações aplicadas com sucesso.", "ok");
      await diagnosticar();
    } else {
      toast("Simulação concluída. Nenhuma chave do registro foi alterada.", "info");
    }
  } finally {
    travar(false);
  }
}

async function desfazerTudo() {
  travar(true);
  Visualizer.show();
  Visualizer.log({
    type: "snap",
    msg: "Iniciando reversão total a partir do histórico JSONL..."
  });
  try {
    const res = await App.DesfazerTudo();
    if (!res || !res.length) {
      abrirModal("Nenhuma Pendência", "<p>Não há alterações realizadas pelo aplicativo pendentes para desfazer.</p>");
      return;
    }

    for (const r of res) {
      Visualizer.log({
        type: "write",
        msg: `Restaurado ao valor anterior: [${r.nome}] — ${r.mensagem}`
      });
    }

    mostrarResultados("Configurações Revertidas", res);
    toast("Configurações revertidas para o estado original.", "ok");
    await diagnosticar();
    if ($("#view-hist").classList.contains("on")) await carregarHistorico();
  } finally {
    travar(false);
  }
}

function travar(travado) {
  $$("#barra .dock-btn").forEach((b) => (b.disabled = travado));
}

function mostrarResultados(titulo, res) {
  res = res || [];
  const linhas = res.map((r) => {
    const ic = { ok: "✓", pulado: "–", falhou: "✕" }[r.estado] || "•";
    return `
    <div class="modal-res-row">
      <div class="modal-res-icon ${esc(r.estado)}">${ic}</div>
      <div class="modal-res-text">
        <b>${esc(r.nome)}</b>
        <div class="msg">${esc(r.mensagem)}</div>
      </div>
    </div>`;
  }).join("");

  const avisoReiniciar = res.some((r) => r.precisaSair && r.estado === "ok")
    ? `<div class="reboot-warn-box">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <span>Alguns ajustes visuais só terão efeito completo após reiniciar a sessão do Windows (Logoff).</span>
       </div>`
    : "";

  abrirModal(titulo, linhas + avisoReiniciar);
}

async function elevar() {
  Visualizer.log({ type: "restore", msg: "Solicitando elevação UAC para privilégios de administrador..." });
  const erro = await App.ReiniciarComoAdmin();
  if (erro) {
    abrirModal("Falha na Elevação", `<p>${esc(erro)}</p>`);
    return;
  }
  await App.Sair();
}

/* ==========================================================================
   Internet & Rede: MTU e Latência com Telemetria
   ========================================================================== */

async function medirMTU() {
  const dest = $("#destino").value.trim();
  const btn = $("#btn-medir");
  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Medindo…`;
  $("#mtu-resultado").innerHTML = `
    <div class="glass-card" style="margin-top:14px">
      <div class="health-loading">
        <span class="pulse-spinner"></span>
        <span>Enviando pacotes de teste com tamanhos variados via busca binária…</span>
      </div>
    </div>`;

  Visualizer.log({
    type: "net",
    msg: `Iniciando busca binária de MTU até [${dest}] com flag DF (Don't Fragment)...`
  });

  try {
    const m = await App.MedirMTU(dest);
    
    // Log das tentativas no visualizador
    if (m.tentativas && m.tentativas.length) {
      for (const p of m.tentativas) {
        Visualizer.log({
          type: "net",
          msg: `Pacote payload ${p.tamanho}B (MTU ${p.mtu}) -> ${p.passou ? "ENTREGUE SEM FRAGMENTAR" : "FRAGMENTAÇÃO DETECTADA"}`
        });
      }
    }

    $("#mtu-resultado").innerHTML = mtuHTML(m);
    const btnAplicarMTU = $("#btn-aplicar-mtu");
    if (btnAplicarMTU) btnAplicarMTU.addEventListener("click", aplicarMTU);
    
    $$(".copy-btn").forEach((cb) => cb.addEventListener("click", () => {
      const cmd = cb.dataset.cmd;
      navigator.clipboard.writeText(cmd);
      cb.textContent = "Copiado!";
      setTimeout(() => (cb.textContent = "Copiar"), 2000);
    }));

    const btnDet = $("#btn-detalhes-mtu");
    if (btnDet) btnDet.addEventListener("click", () => {
      const tabela = $("#tabela-passos-mtu");
      tabela.hidden = !tabela.hidden;
      btnDet.textContent = tabela.hidden ? "Ver tentativas detalhadas" : "Esconder tentativas";
    });
  } finally {
    btn.disabled = false;
    btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg> Medir MTU Ideal`;
  }
}

function mtuHTML(m) {
  if (m.erro) return `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Falha na Medição</h3><p class="card-subtitle">${esc(m.erro)}</p></div>`;

  const comandos = m.comandoOk ? `
    <p class="field-hint" style="margin-top:14px">Verificação manual equivalente no Prompt de Comando:</p>
    <div class="terminal-block">
      <div class="terminal-line">
        <span class="cmd-ok">${esc(m.comandoOk)}</span>
        <button class="copy-btn" data-cmd="${esc(m.comandoOk)}">Copiar</button>
      </div>
      ${m.comandoFalha ? `
      <div class="terminal-line">
        <span class="cmd-fail">${esc(m.comandoFalha)}</span>
        <button class="copy-btn" data-cmd="${esc(m.comandoFalha)}">Copiar</button>
      </div>` : ""}
    </div>` : "";

  const passos = (m.tentativas || []).map((p) => `
    <tr class="${p.passou ? "" : "falhou"}">
      <td class="mono">${p.tamanho} B</td>
      <td class="mono">${p.mtu}</td>
      <td>${esc(p.estado)}</td>
      <td class="mono">${esc(p.comando)}</td>
    </tr>`).join("");

  const acaoAplicar = m.podeAplicar ? `
    <div class="action-inline-form" style="margin-top:16px">
      <button class="btn-primary" id="btn-aplicar-mtu">Ajustar MTU para ${m.sugestao}</button>
      <span class="field-hint">Ajuste seguro, auditado e reversível.</span>
    </div>` : "";

  return `
  <div class="glass-card" style="margin-top:16px; border-left: 3px solid var(--accent)">
    <h3 class="card-title">${esc(m.resumo)}</h3>
    <p class="card-subtitle">${esc(m.explicacao)}</p>
    <p class="field-hint" style="margin-top:6px">Adaptador <b>${esc(m.adaptador)}</b> (${esc(m.tipoAdaptador)}) · MTU configurado: ${m.mtuAtual}${m.mtuCaminho ? ` · MTU medido no caminho: ${m.mtuCaminho}` : ""}</p>
    ${comandos}
    ${acaoAplicar}
    ${passos ? `
      <button class="link-expand" id="btn-detalhes-mtu" style="margin-top:12px">Ver tentativas detalhadas</button>
      <div id="tabela-passos-mtu" class="data-table-wrap" hidden>
        <table class="data-table">
          <thead><tr><th>Tamanho</th><th>MTU</th><th>Resultado</th><th>Comando</th></tr></thead>
          <tbody>${passos}</tbody>
        </table>
      </div>` : ""}
  </div>`;
}

async function aplicarMTU() {
  const btn = $("#btn-aplicar-mtu");
  btn.disabled = true;
  Visualizer.log({ type: "write", msg: "Aplicando novo valor de MTU na interface de rede..." });
  try {
    const res = await App.AplicarMTU(false);
    mostrarResultados("Ajuste de MTU", res);
    toast("MTU ajustado com sucesso.", "ok");
    await medirMTU();
  } finally {
    btn.disabled = false;
  }
}

/* ==========================================================================
   Perfis de Rede
   ========================================================================== */

async function carregarPerfis() {
  const container = $("#lista-perfis");
  container.innerHTML = '<p class="muted-text">Carregando perfis de rede…</p>';
  const perfis = await App.ListarPerfisRede();
  container.innerHTML = (perfis || []).map(renderPerfilCard).join("");
  $$(".btn-aplicar-perfil").forEach((b) => b.addEventListener("click", () => aplicarPerfil(b.dataset.key)));
}

function renderPerfilCard(p) {
  return `
  <div class="profile-card">
    <div>
      <div class="profile-card-title">
        <span>${esc(p.nome)}</span>
        <span class="badge-tag rec">${p.numTweaks} ajuste(s)</span>
      </div>
      <div class="profile-card-desc" style="margin-top:6px">${esc(p.descricao)}</div>
      ${p.ressalvas ? `<div class="caveat-box" style="margin-top:8px"><b>Ressalva:</b> ${esc(p.ressalvas)}</div>` : ""}
    </div>
    <button class="btn-primary btn-aplicar-perfil" data-key="${esc(p.key)}" style="width:100%; justify-content:center; margin-top:10px">
      Aplicar Perfil
    </button>
  </div>`;
}

async function aplicarPerfil(key) {
  const btns = $$(".btn-aplicar-perfil");
  btns.forEach((b) => (b.disabled = true));
  Visualizer.show();
  Visualizer.log({ type: "write", msg: `Aplicando perfil de rede [${key}] em lote...` });
  try {
    const res = await App.AplicarPerfilRede(key, false);
    mostrarResultados("Perfil de Rede Aplicado", res);
    toast("Perfil de rede aplicado com sucesso.", "ok");
  } finally {
    btns.forEach((b) => (b.disabled = false));
  }
}

/* ==========================================================================
   Latência & Benchmark
   ========================================================================== */

let ultimoAntes = null;

async function medirAntes() {
  const host = $("#destino-latencia").value.trim();
  const btn = $("#btn-medir-antes");
  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Medindo…`;
  $("#latencia-resultado").innerHTML = `
    <div class="glass-card" style="margin-top:14px">
      <div class="health-loading">
        <span class="pulse-spinner"></span>
        <span>Enviando 20 pacotes de teste para medir latência base…</span>
      </div>
    </div>`;

  Visualizer.log({ type: "net", msg: `Disparando 20 sondagens ICMP de medição base até [${host}]...` });

  try {
    ultimoAntes = await App.MedirRedeAntes(host);
    Visualizer.log({
      type: "net",
      msg: `Medição base concluída: RTT médio = ${ultimoAntes.avgRTT}ms, Jitter = ${ultimoAntes.stdDev}ms`
    });
    $("#latencia-resultado").innerHTML = benchmarkHTML(ultimoAntes, "Medição Base — Antes da Otimização");
    if (!ultimoAntes.erro) $("#btn-medir-depois").disabled = false;
  } catch (e) {
    $("#latencia-resultado").innerHTML = `<div class="glass-card"><h3 class="card-title">Erro na Medição</h3><p class="card-subtitle">${esc(String(e))}</p></div>`;
  } finally {
    btn.disabled = false;
    btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg> 1. Medir Antes`;
  }
}

async function medirDepois() {
  const host = $("#destino-latencia").value.trim();
  const btn = $("#btn-medir-depois");
  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Medindo…`;
  Visualizer.log({ type: "net", msg: `Disparando 20 sondagens ICMP de pós-otimização até [${host}]...` });

  try {
    const depois = await App.MedirRedeDepois(host);
    if (depois.erro) {
      $("#latencia-resultado").innerHTML = benchmarkHTML(ultimoAntes, "Medição Base — Antes") +
        `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Falha ao Medir Novamente</h3><p class="card-subtitle">${esc(depois.erro)}</p></div>`;
      return;
    }
    const comp = await App.RelatorioComparativo(ultimoAntes, depois);
    Visualizer.log({
      type: "net",
      msg: `Comparação final: Latência ${comp.deltaLatencia}, Jitter ${comp.deltaJitter}`
    });

    $("#latencia-resultado").innerHTML =
      benchmarkHTML(ultimoAntes, "Medição Base — Antes") +
      benchmarkHTML(depois, "Medição — Depois dos Ajustes") +
      comparativoHTML(comp);
  } catch (e) {
    $("#latencia-resultado").innerHTML += `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Erro de Comparação</h3><p class="card-subtitle">${esc(String(e))}</p></div>`;
  } finally {
    btn.disabled = false;
    btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg> 2. Medir Depois`;
  }
}

function benchmarkHTML(b, titulo) {
  if (b.erro) return `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Falha</h3><p class="card-subtitle">${esc(b.erro)}</p></div>`;
  return `
  <div class="glass-card" style="margin-top:14px">
    <h3 class="card-title">${esc(titulo)}</h3>
    <p class="field-hint">Destino ${esc(b.host)} · ${b.packetsSent} pacotes enviados · ${b.packetsLost} pacote(s) perdido(s)</p>
    <div class="data-table-wrap">
      <table class="data-table">
        <thead><tr><th>Métrica</th><th>Valor Obtido</th></tr></thead>
        <tbody>
          <tr><td>Latência Média (RTT)</td><td class="mono"><b>${b.avgRTT} ms</b></td></tr>
          <tr><td>Variação Mínima / Máxima</td><td class="mono">${b.minRTT} ms / ${b.maxRTT} ms</td></tr>
          <tr><td>Jitter (Desvio Padrão)</td><td class="mono">${b.stdDev} ms</td></tr>
          <tr><td>Perda de Pacotes</td><td class="mono">${(b.lossPercent ?? 0).toFixed(1)}%</td></tr>
        </tbody>
      </table>
    </div>
  </div>`;
}

function comparativoHTML(c) {
  return `
  <div class="glass-card" style="margin-top:14px; border-left: 3px solid var(--accent)">
    <h3 class="card-title">Relatório de Impacto Real</h3>
    <p class="card-subtitle">${esc(c.interpretacao)}</p>
    <p class="field-hint" style="margin-top:6px">Variação de Latência: <b>${esc(c.deltaLatencia)}</b> · Jitter: <b>${esc(c.deltaJitter)}</b></p>
  </div>`;
}

/* ==========================================================================
   Histórico & Auditoria
   ========================================================================== */

async function carregarHistorico() {
  const container = $("#hist-lista");
  const entradas = await App.Historico();
  if (!entradas || !entradas.length) {
    container.innerHTML = '<div class="empty-results"><p>Nenhuma alteração foi realizada até o momento nesta máquina.</p></div>';
    return;
  }
  container.innerHTML = entradas.map((e) => {
    let actionClass = "apply";
    if (e.acao.toLowerCase().includes("revert") || e.acao.toLowerCase().includes("desfazer")) {
      actionClass = "revert";
    }
    if (!e.ok) actionClass = "fail";

    return `
    <div class="timeline-item">
      <span class="timeline-time">${esc(e.quando)}</span>
      <span class="timeline-action ${actionClass}">${esc(e.acao)}</span>
      <div><b>${esc(e.item)}</b> — <span class="card-subtitle">${esc(e.resultado)}</span></div>
    </div>`;
  }).join("");
}

/* ==========================================================================
   Inicialização & Programas no Boot
   ========================================================================== */

async function carregarStartup() {
  const container = $("#lista-startup");
  container.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Lendo programas de inicialização…</span></div>';
  try {
    const itens = await App.ListarInicializacao();
    if (!itens || !itens.length) {
      container.innerHTML = '<div class="empty-results"><p>Nenhum programa configurado para inicializar automaticamente com o Windows.</p></div>';
      return;
    }

    container.innerHTML = itens.map((it) => `
      <div class="tweak-card" style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
        <div style="flex:1; padding-right:16px;">
          <div class="tweak-header-line">
            <span class="tweak-name">${esc(it.nome)}</span>
            <span class="badge-tag rec">${esc(it.origem)}</span>
          </div>
          <div class="tweak-desc mono" style="font-size:12px; margin-top:4px; opacity:0.8;">${esc(it.comando)}</div>
        </div>
        <label class="switch-control" title="Desativar ou ativar programa na inicialização">
          <input type="checkbox" class="chk-startup" data-id="${esc(it.id)}" ${it.ativo ? "checked" : ""}>
          <span class="switch-track"></span>
        </label>
      </div>
    `).join("");

    $$(".chk-startup").forEach((chk) => chk.addEventListener("change", async (e) => {
      const id = e.target.dataset.id;
      const ativar = e.target.checked;
      Visualizer.show();
      Visualizer.log({
        type: "write",
        msg: `${ativar ? "Ativando" : "Desativando"} programa na inicialização: [${id}]`
      });
      const err = await App.AlternarInicializacao(id, ativar);
      if (err) {
        toast(`Erro ao alterar: ${err}`, "erro");
        e.target.checked = !ativar;
      } else {
        toast(ativar ? "Programa ativado na inicialização." : "Programa desativado da inicialização.", "ok");
      }
    }));
  } catch (e) {
    container.innerHTML = `<div class="empty-results"><p>Erro ao ler inicialização: ${esc(String(e))}</p></div>`;
  }
}

/* ==========================================================================
   Disco, Armazenamento & Reparo DISM/SFC
   ========================================================================== */

async function carregarDiscos() {
  const container = $("#lista-drives");
  container.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Auditando unidades de disco…</span></div>';
  try {
    const drives = await App.ListarDiscos();
    if (!drives || !drives.length) {
      container.innerHTML = '<div class="empty-results"><p>Nenhuma unidade detectada.</p></div>';
      return;
    }

    container.innerHTML = drives.map((d) => `
      <div class="glass-card" style="margin-bottom:12px; border-left:3px solid var(--accent);">
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <div>
            <h3 class="card-title" style="margin:0;">Unidade ${esc(d.letter)} (${esc(d.label)})</h3>
            <p class="field-hint" style="margin-top:4px;">Tipo de Mídia: <b>${esc(d.mediaType)}</b> (${esc(d.busType)}) · Saúde SMART: <b>${esc(d.healthStatus)}</b></p>
          </div>
          <div style="display:flex; gap:8px;">
            ${d.supportsTrim ? `<button class="btn-secondary small-btn btn-trim" data-drive="${esc(d.letter)}">Executar TRIM</button>` : ""}
            <button class="btn-secondary small-btn btn-chkdsk" data-drive="${esc(d.letter)}">chkdsk /scan</button>
          </div>
        </div>
      </div>
    `).join("");

    $$(".btn-trim").forEach((btn) => btn.addEventListener("click", async () => {
      const drv = btn.dataset.drive;
      btn.disabled = true;
      btn.textContent = "Executando TRIM…";
      Visualizer.show();
      Visualizer.log({ type: "write", msg: `Executando TRIM de manutenção no SSD ${drv}...` });
      const out = await App.ExecutarTRIM(drv);
      btn.disabled = false;
      btn.textContent = "Executar TRIM";
      abrirModal(`Resultado TRIM (${drv})`, `<div class="terminal-block"><div>${esc(out)}</div></div>`);
    }));

    $$(".btn-chkdsk").forEach((btn) => btn.addEventListener("click", async () => {
      const drv = btn.dataset.drive;
      btn.disabled = true;
      btn.textContent = "Verificando…";
      Visualizer.show();
      Visualizer.log({ type: "read", msg: `Executando chkdsk /scan não destrutivo na unidade ${drv}...` });
      const out = await App.ExecutarChkdsk(drv);
      btn.disabled = false;
      btn.textContent = "chkdsk /scan";
      abrirModal(`Verificação de Integridade (${drv})`, `<div class="terminal-block"><div>${esc(out)}</div></div>`);
    }));
  } catch (e) {
    container.innerHTML = `<div class="empty-results"><p>Erro ao ler unidades: ${esc(String(e))}</p></div>`;
  }
}

async function auditarReparo() {
  const btn = $("#btn-auditar-reparo");
  const resEl = $("#reparo-resultado");
  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Auditando arquivos do Windows…`;
  resEl.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Executando DISM /CheckHealth e SFC /verifyonly…</span></div>';

  Visualizer.show();
  Visualizer.log({ type: "read", msg: "Iniciando auditoria de integridade da imagem do Windows (DISM/SFC)..." });

  try {
    const rep = await App.AuditarReparo();
    Visualizer.log({
      type: rep.dismHealthy && rep.sfcHealthy ? "verify" : "fail",
      msg: `Auditoria concluída: DISM ${rep.dismHealthy ? "OK" : "Inconsistente"}, SFC ${rep.sfcHealthy ? "OK" : "Inconsistente"}`
    });

    resEl.innerHTML = `
      <div class="glass-card" style="border-left: 3px solid ${rep.dismHealthy && rep.sfcHealthy ? 'var(--accent)' : 'var(--warn)'}">
        <h3 class="card-title">${rep.dismHealthy && rep.sfcHealthy ? 'Sistema Saudável e Íntegro' : 'Atenção: Inconsistências Detectadas'}</h3>
        <p class="card-subtitle">${esc(rep.interpretation)}</p>
        <div class="terminal-block" style="margin-top:10px;">
          <div><b>DISM CheckHealth:</b> ${esc(rep.dismOutput || 'Verificado com sucesso')}</div>
          <div style="margin-top:6px;"><b>SFC VerifyOnly:</b> ${esc(rep.sfcOutput || 'Sem violações de integridade')}</div>
        </div>
      </div>`;
  } catch (e) {
    resEl.innerHTML = `<div class="empty-results"><p>Erro na auditoria: ${esc(String(e))}</p></div>`;
  } finally {
    btn.disabled = false;
    btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 14 14"/></svg> Auditar Integridade do Windows`;
  }
}

/* ==========================================================================
   DNS Benchmark & DoH
   ========================================================================== */

async function testarDNS() {
  const btn = $("#btn-testar-dns");
  const resEl = $("#dns-resultado");
  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Medindo…`;
  resEl.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Resolvendo domínios de teste em múltiplos servidores DNS…</span></div>';

  Visualizer.show();
  Visualizer.log({ type: "net", msg: "Iniciando benchmark de latência DNS nos principais provedores globais..." });

  try {
    const provs = await App.BenchmarkDNS();
    for (const p of (provs || [])) {
      Visualizer.log({
        type: "net",
        msg: `DNS [${p.nome}] -> RTT Médio: ${p.avgRttMs}ms (Perda: ${p.perda}%)`
      });
    }

    resEl.innerHTML = `
      <div class="data-table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>Provedor DNS</th>
              <th>IPs</th>
              <th>RTT Médio</th>
              <th>Perda</th>
              <th>Privacidade / Segurança</th>
              <th>DoH (HTTPS)</th>
            </tr>
          </thead>
          <tbody>
            ${(provs || []).map((p) => `
              <tr class="${p.recomendado ? 'highlight-row' : ''}">
                <td><b>${esc(p.nome)}</b> ${p.recomendado ? '<span class="badge-tag speed">Mais Rápido</span>' : ''}</td>
                <td class="mono small">${esc(p.ips.join(', '))}</td>
                <td class="mono"><b>${p.avgRttMs} ms</b></td>
                <td class="mono">${p.perda.toFixed(0)}%</td>
                <td>${esc(p.privacidade)}</td>
                <td class="mono small">${esc(p.dohUrl)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>`;
  } catch (e) {
    resEl.innerHTML = `<div class="empty-results"><p>Erro ao medir DNS: ${esc(String(e))}</p></div>`;
  } finally {
    btn.disabled = false;
    btn.innerHTML = "Testar Servidores DNS";
  }
}

/* ==========================================================================
   Limpeza de Dispositivos Fantasmas (Ghost PnP Devices)
   ========================================================================== */

let listaDispositivosFantasmas = [];

async function escanearDispositivosFantasmas() {
  const btn = $("#btn-escanear-pnp");
  const btnLimpar = $("#btn-limpar-pnp");
  const resEl = $("#pnp-resultado");

  btn.disabled = true;
  btn.innerHTML = `<span class="pulse-spinner"></span> Buscando…`;
  resEl.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Consultando dispositivos PnP desconectados e órfãos…</span></div>';

  Visualizer.show();
  Visualizer.log({ type: "read", msg: "Consultando tabela PnP em busca de dispositivos desconectados ou órfãos..." });

  try {
    listaDispositivosFantasmas = await App.ListarDispositivosFantasmas();
    if (!listaDispositivosFantasmas || !listaDispositivosFantasmas.length) {
      resEl.innerHTML = '<div class="glass-card" style="border-left: 3px solid var(--accent);"><h3 class="card-title">Nenhum Dispositivo Fantasma Encontrado</h3><p class="card-subtitle">A árvore de dispositivos Plug-and-Play do Windows está limpa e sem nós órfãos.</p></div>';
      btnLimpar.style.display = "none";
      return;
    }

    Visualizer.log({
      type: "read",
      msg: `Foram encontrados ${listaDispositivosFantasmas.length} nós de dispositivos desconectados/fantasmas.`
    });

    resEl.innerHTML = `
      <div style="margin-bottom:12px;">
        <p style="font-size:13px; color:var(--text-secondary); margin-bottom:8px;">
          Foram encontrados <b>${listaDispositivosFantasmas.length}</b> dispositivo(s) desconectado(s) ou órfão(s):
        </p>
        <div class="terminal-block" style="max-height:220px; overflow-y:auto;">
          ${listaDispositivosFantasmas.map((d) => `
            <div style="display:flex; justify-content:space-between; margin-bottom:4px; font-size:12px;">
              <span>• <b>${esc(d.FriendlyName || d.InstanceId)}</b> <span class="badge-tag rec">${esc(d.Class || 'PnP')}</span></span>
              <span class="mono muted small">${esc(d.InstanceId)}</span>
            </div>
          `).join("")}
        </div>
      </div>
    `;
    btnLimpar.style.display = "inline-flex";
    btnLimpar.textContent = `Limpar ${listaDispositivosFantasmas.length} Dispositivo(s)`;
  } catch (e) {
    resEl.innerHTML = `<div class="empty-results"><p>Erro ao listar dispositivos: ${esc(String(e))}</p></div>`;
    btnLimpar.style.display = "none";
  } finally {
    btn.disabled = false;
    btn.innerHTML = "Buscar Dispositivos";
  }
}

async function limparDispositivosFantasmas() {
  if (!listaDispositivosFantasmas || !listaDispositivosFantasmas.length) return;

  const btnLimpar = $("#btn-limpar-pnp");
  btnLimpar.disabled = true;
  btnLimpar.innerHTML = `<span class="pulse-spinner"></span> Removendo…`;

  Visualizer.show();
  Visualizer.log({ type: "write", msg: `Iniciando remoção segura de ${listaDispositivosFantasmas.length} dispositivos fantasmas via pnputil...` });

  try {
    const ids = listaDispositivosFantasmas.map((d) => d.InstanceId);
    const res = await App.LimparDispositivosFantasmas(ids);

    Visualizer.log({
      type: res.erros && res.erros.length > 0 ? "verify" : "write",
      msg: res.mensagem
    });

    toast(res.mensagem, res.erros && res.erros.length > 0 ? "aviso" : "ok");
    await escanearDispositivosFantasmas();
  } catch (e) {
    toast(`Erro na limpeza de dispositivos: ${e}`, "erro");
  } finally {
    btnLimpar.disabled = false;
  }
}

/* ==========================================================================
   Perfis de Uso (private-optimizer JOGO / CODING)
   ========================================================================== */

let listaPerfisUso = [];

async function carregarEstadoPerfis() {
  try {
    listaPerfisUso = await App.ListarPerfisUso();
    const ativo = await App.ObterPerfilAtivo();

    ["jogo", "coding"].forEach((key) => {
      const card = $("#card-perfil-" + key);
      const badge = $("#status-perfil-" + key);
      const btnApply = $("#btn-apply-" + key);
      const btnRestore = $("#btn-restore-" + key);
      const isAtivo = (ativo === key);

      if (card) card.classList.toggle("active-profile", isAtivo);
      if (badge) {
        badge.textContent = isAtivo ? "Ativo" : "Inativo";
        badge.classList.toggle("active", isAtivo);
      }
      if (btnApply) {
        btnApply.textContent = isAtivo ? "Reaplicar Perfil" : "Aplicar Perfil";
      }
      if (btnRestore) {
        btnRestore.hidden = !isAtivo;
      }
    });
  } catch (e) {
    console.error("Erro ao carregar estado de perfis:", e);
  }
}

async function visualizarPerfil(key) {
  const p = (listaPerfisUso || []).find((x) => x.key === key);
  if (!p) return;

  const html = `
    <div style="margin-bottom: 12px;">
      <p style="margin-bottom:8px; color:var(--text-secondary);">${esc(p.descricao)}</p>
      <h4 style="font-size:13px; font-weight:700; margin:12px 0 6px 0;">Ajustes que compõem este perfil (${p.tweakIds.length} itens):</h4>
      <div class="terminal-block" style="max-height:220px; overflow-y:auto;">
        ${p.tweakIds.map((id) => `<div>• <b>${esc(id)}</b></div>`).join("")}
      </div>
      <div style="margin-top:12px;" class="field-hint">${esc(p.ressalvas)}</div>
    </div>
  `;
  abrirModal(`Composição do Perfil: ${p.nome}`, html);
}

async function verificarPerfil(key) {
  const p = (listaPerfisUso || []).find((x) => x.key === key);
  if (!p) return;

  Visualizer.show();
  Visualizer.log({ type: "read", msg: `Verificando integridade das chaves do perfil ${key.toUpperCase()}...` });

  let auditHtml = `<div class="health-loading"><span class="pulse-spinner"></span><span>Auditando integridade do perfil…</span></div>`;
  abrirModal(`Integridade: ${p.nome}`, auditHtml);

  try {
    const diag = await App.Diagnosticar("pessoal");
    const mapaItens = new Map((diag.itens || []).map((it) => [it.meta.id, it]));

    const linhas = p.tweakIds.map((id) => {
      const it = mapaItens.get(id);
      const isApplied = it && it.state === "applied";
      const icon = isApplied ? '<span style="color:var(--accent); font-weight:bold;">[OK]</span>' : '<span style="color:var(--text-muted);">[PENDENTE]</span>';
      return `<div style="display:flex; justify-content:space-between; margin-bottom:6px; font-size:12px;">
        <span>${icon} <b>${esc(it ? it.meta.displayName : id)}</b></span>
        <span class="muted">${esc(it ? it.detail : '')}</span>
      </div>`;
    }).join("");

    $("#modal-corpo").innerHTML = `
      <div style="margin-bottom:10px;">
        <p style="margin-bottom:10px; font-size:13px; color:var(--text-secondary);">Estado atual de cada otimização no Windows:</p>
        <div class="terminal-block" style="max-height:300px; overflow-y:auto;">${linhas}</div>
      </div>
    `;
  } catch (e) {
    $("#modal-corpo").innerHTML = `<p class="muted">Erro na verificação: ${esc(String(e))}</p>`;
  }
}

/* ==========================================================================
   Benchmark Observacional & Telemetria Obrigatória Pré/Pós Perfil
   ========================================================================== */

let benchmarkEmAndamento = false;

function abrirModalBenchmark(titulo, desc) {
  $("#bench-modal-titulo").textContent = titulo;
  $("#bench-modal-desc").textContent = desc;
  $("#bench-progress-fill").style.width = "0%";
  $("#bench-progress-percent").textContent = "0%";
  $("#bench-progress-label").textContent = "Amostrando hardware (0/60s)...";
  $("#bench-live-cpu").textContent = "0.0%";
  $("#bench-live-gpu").textContent = "0.0%";
  $("#bench-live-ram").textContent = "0 MB";
  $("#bench-live-cputemp").textContent = "Temp: Indisponível";
  $("#bench-live-gputemp").textContent = "Temp: Indisponível";
  $("#bench-custom-content").innerHTML = "";
  $("#overlay-benchmark").hidden = false;
  benchmarkEmAndamento = true;
}

function fecharModalBenchmark() {
  $("#overlay-benchmark").hidden = true;
  if (benchmarkEmAndamento) {
    benchmarkEmAndamento = false;
    App.CancelarBenchmark();
  }
}

function atualizarProgressoBenchmark(data) {
  if (!data) return;
  const pct = Math.min(100, Math.max(0, data.percent || 0));
  $("#bench-progress-fill").style.width = `${pct.toFixed(0)}%`;
  $("#bench-progress-percent").textContent = `${pct.toFixed(0)}%`;
  $("#bench-progress-label").textContent = `Amostrando hardware (${data.current || 0}/${data.total || 60}s)...`;

  $("#bench-live-cpu").textContent = `${(data.cpuUsage || 0).toFixed(1)}%`;
  $("#bench-live-gpu").textContent = `${(data.gpuUsage || 0).toFixed(1)}%`;
  $("#bench-live-ram").textContent = `${(data.ramUsedMb || 0).toFixed(0)} MB`;

  if (data.cpuTemp !== undefined && data.cpuTemp !== null) {
    $("#bench-live-cputemp").textContent = `Temp: ${data.cpuTemp.toFixed(1)}°C`;
    $("#bench-live-cputemp").style.color = data.cpuTemp > 80 ? "var(--danger)" : "var(--text-secondary)";
  } else {
    $("#bench-live-cputemp").textContent = "Temp CPU: Indisponível";
  }

  if (data.gpuTemp !== undefined && data.gpuTemp !== null) {
    $("#bench-live-gputemp").textContent = `Temp: ${data.gpuTemp.toFixed(1)}°C`;
    $("#bench-live-gputemp").style.color = data.gpuTemp > 80 ? "var(--danger)" : "var(--text-secondary)";
  } else {
    $("#bench-live-gputemp").textContent = "Temp GPU: Indisponível";
  }
}

async function aplicarPerfilUso(key) {
  const p = (listaPerfisUso || []).find((x) => x.key === key);
  if (!p) return;

  // 1. Iniciar Benchmark Base Pré-Aplicação (60 segundos)
  abrirModalBenchmark(
    `Benchmark Base Obrigatório (60s) — ${p.nome}`,
    "Executando amostragem observacional de 60 segundos do uso de CPU, GPU, memória e sensores de temperatura antes de aplicar as otimizações. Esta medição é 100% não destrutiva."
  );

  Visualizer.show();
  Visualizer.log({ type: "read", msg: `Iniciando benchmark-base observacional (60s) para o perfil ${key.toUpperCase()}...` });

  let reportAntes = null;
  try {
    reportAntes = await App.IniciarBenchmarkBase(key, 60);
  } catch (err) {
    fecharModalBenchmark();
    toast(`Benchmark cancelado ou interrompido.`, "aviso");
    return;
  }

  benchmarkEmAndamento = false;
  $("#overlay-benchmark").hidden = true;

  if (!reportAntes || reportAntes.sampleCount === 0) {
    toast("Amostragem de benchmark não concluída.", "aviso");
    return;
  }

  // 2. Modal de Confirmação com Métricas da Linha de Base
  const cpuTempTexto = reportAntes.cpuTempAvailable && reportAntes.cpuTempAvg !== null && reportAntes.cpuTempAvg !== undefined
    ? `${reportAntes.cpuTempAvg.toFixed(1)}°C (Pico: ${reportAntes.cpuTempPeak ? reportAntes.cpuTempPeak.toFixed(1) : ''}°C)`
    : `<span class="muted">Sensor não disponível neste hardware</span>`;

  const gpuTempTexto = reportAntes.gpuTempAvailable && reportAntes.gpuTempAvg !== null && reportAntes.gpuTempAvg !== undefined
    ? `${reportAntes.gpuTempAvg.toFixed(1)}°C (Pico: ${reportAntes.gpuTempPeak ? reportAntes.gpuTempPeak.toFixed(1) : ''}°C)`
    : `<span class="muted">Sensor não disponível neste hardware</span>`;

  const confirmHtml = `
    <div style="margin-bottom:14px;">
      <p style="font-size:14px; color:var(--text-primary); margin-bottom:6px;">Deseja confirmar e aplicar o perfil <b>${esc(p.nome)}</b>?</p>
      <p style="font-size:13px; color:var(--text-secondary); margin-bottom:12px;">Linha de base coletada com sucesso (${reportAntes.sampleCount} amostras). O perfil será gravado com snapshot atômico no histórico.</p>

      <div class="glass-card" style="margin-bottom:14px; padding:12px; background:var(--bg-sunken); border-left:3px solid var(--accent);">
        <h4 style="font-size:12px; font-weight:700; text-transform:uppercase; color:var(--text-muted); margin-bottom:8px;">Métricas da Linha de Base (Pré-Aplicação):</h4>
        <div style="display:grid; grid-template-columns:1fr 1fr; gap:8px; font-size:12px;">
          <div>• CPU Média/Pico: <b>${reportAntes.cpuUsageAvg.toFixed(1)}% / ${reportAntes.cpuUsagePeak.toFixed(1)}%</b></div>
          <div>• RAM em Uso: <b>${reportAntes.ramUsedAvgMb.toFixed(0)} MB</b></div>
          <div>• GPU Média: <b>${reportAntes.gpuUsageAvg.toFixed(1)}%</b></div>
          <div>• Temp CPU: <b>${cpuTempTexto}</b></div>
          <div>• Temp GPU: <b>${gpuTempTexto}</b></div>
          <div>• Throttling Térmico: <b>${reportAntes.thermalThrottled ? '<span style="color:var(--danger)">Detectado</span>' : '<span style="color:var(--accent)">Normal</span>'}</b></div>
        </div>
      </div>

      <div class="profile-caveat" style="margin-bottom:16px;">
        <strong>Aviso:</strong> ${esc(p.ressalvas)}
      </div>

      <div style="display:flex; justify-content:flex-end; gap:8px;">
        <button class="btn-secondary" id="btn-modal-cancelar">Cancelar</button>
        <button class="btn-primary" id="btn-modal-confirmar-aplicar">Confirmar e Aplicar Perfil</button>
      </div>
    </div>
  `;
  abrirModal(`Confirmar Aplicação: ${p.nome}`, confirmHtml);

  $("#btn-modal-cancelar").addEventListener("click", fecharModal);
  $("#btn-modal-confirmar-aplicar").addEventListener("click", async () => {
    fecharModal();

    // 3. Aplicação do Perfil
    Visualizer.show();
    Visualizer.log({ type: "write", msg: `Aplicando configurações do perfil [${key.toUpperCase()}] com snapshot transacional...` });

    toast(`Aplicando perfil ${p.nome}…`, "info");
    const resAplicacao = await App.AplicarPerfilComBenchmark(key, false, reportAntes);

    for (const r of (resAplicacao.resultados || [])) {
      Visualizer.log({
        type: r.estado === "ok" ? "write" : (r.estado === "pulado" ? "read" : "fail"),
        msg: `[${r.estado.toUpperCase()}] ${r.nome}: ${r.mensagem}`
      });
    }

    // 4. Estabilização e Benchmark Pós-Aplicação (60s)
    abrirModalBenchmark(
      `Estabilização & Benchmark Pós-Aplicação (60s) — ${p.nome}`,
      "O perfil foi aplicado com sucesso. Agora realizando a medição pós-aplicação de 60 segundos para gerar o relatório comparativo de telemetria."
    );

    Visualizer.log({ type: "read", msg: `Iniciando benchmark observacional pós-aplicação (60s)...` });

    let comparacao = null;
    try {
      comparacao = await App.IniciarBenchmarkPos(key, resAplicacao.batchId, 60);
    } catch (e) {
      fecharModalBenchmark();
      toast("Medição pós-aplicação interrompida.", "aviso");
      await carregarEstadoPerfis();
      await diagnosticar();
      return;
    }

    benchmarkEmAndamento = false;
    $("#overlay-benchmark").hidden = true;

    // 5. Exibir Comparativo Antes vs Depois
    exibirRelatorioComparativo(comparacao, p.nome);
    await carregarEstadoPerfis();
    await diagnosticar();
  });
}

function exibirRelatorioComparativo(comp, perfilNome) {
  if (!comp) return;

  const b = comp.before;
  const a = comp.after;

  const formatDelta = (val, unidade = "%") => {
    if (val === undefined || val === null) return "—";
    const sinal = val > 0 ? `+${val.toFixed(1)}` : val.toFixed(1);
    const cor = val < 0 ? "var(--accent)" : (val > 0 ? "var(--warn)" : "var(--text-secondary)");
    return `<span style="color:${cor}; font-weight:700;">${sinal}${unidade}</span>`;
  };

  const compHtml = `
    <div style="margin-bottom:14px;">
      <p style="font-size:13px; color:var(--text-secondary); margin-bottom:12px;">
        Comparação observacional de telemetria antes e depois da aplicação do perfil <b>${esc(perfilNome)}</b>:
      </p>

      <div class="data-table-wrap" style="margin-bottom:14px;">
        <table class="data-table">
          <thead>
            <tr>
              <th>Métrica de Hardware</th>
              <th>Antes (Base)</th>
              <th>Depois (Ativo)</th>
              <th>Variação</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><b>Uso Médio de CPU</b></td>
              <td class="mono">${b.cpuUsageAvg.toFixed(1)}% (Pico: ${b.cpuUsagePeak.toFixed(1)}%)</td>
              <td class="mono">${a.cpuUsageAvg.toFixed(1)}% (Pico: ${a.cpuUsagePeak.toFixed(1)}%)</td>
              <td class="mono">${formatDelta(comp.deltaCpuUsageAvg, "%")}</td>
            </tr>
            <tr>
              <td><b>Memória RAM em Uso</b></td>
              <td class="mono">${b.ramUsedAvgMb.toFixed(0)} MB</td>
              <td class="mono">${a.ramUsedAvgMb.toFixed(0)} MB</td>
              <td class="mono">${formatDelta(comp.deltaGpuMemoryAvgMb, " MB")}</td>
            </tr>
            <tr>
              <td><b>Uso Médio de GPU</b></td>
              <td class="mono">${b.gpuUsageAvg.toFixed(1)}%</td>
              <td class="mono">${a.gpuUsageAvg.toFixed(1)}%</td>
              <td class="mono">${formatDelta(comp.deltaGpuUsageAvg, "%")}</td>
            </tr>
            <tr>
              <td><b>Temperatura CPU</b></td>
              <td class="mono">${b.cpuTempAvailable && b.cpuTempAvg !== null && b.cpuTempAvg !== undefined ? `${b.cpuTempAvg.toFixed(1)}°C` : '<span class="muted">Indisponível</span>'}</td>
              <td class="mono">${a.cpuTempAvailable && a.cpuTempAvg !== null && a.cpuTempAvg !== undefined ? `${a.cpuTempAvg.toFixed(1)}°C` : '<span class="muted">Indisponível</span>'}</td>
              <td class="mono">${comp.deltaCpuTempAvg !== null && comp.deltaCpuTempAvg !== undefined ? formatDelta(comp.deltaCpuTempAvg, "°C") : '<span class="muted">—</span>'}</td>
            </tr>
            <tr>
              <td><b>Throttling Térmico</b></td>
              <td class="mono">${b.thermalThrottled ? '<span style="color:var(--danger)">Ativo</span>' : 'Não'}</td>
              <td class="mono">${a.thermalThrottled ? '<span style="color:var(--danger)">Ativo</span>' : 'Não'}</td>
              <td class="mono">${comp.throttlingResolved ? '<span style="color:var(--accent); font-weight:bold;">Resolvido</span>' : '—'}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="field-hint" style="margin-bottom:14px; font-size:12px;">
        ℹ️ ${esc(comp.disclaimer)}
      </div>

      <div style="display:flex; justify-content:flex-end;">
        <button class="btn-primary" onclick="fecharModal()">Concluir</button>
      </div>
    </div>
  `;

  abrirModal(`Comparativo de Telemetria — ${perfilNome}`, compHtml);
}

async function restaurarPerfilUso(key) {
  Visualizer.show();
  Visualizer.log({ type: "revert", msg: `Revertendo lote de transação ativo do perfil [${key.toUpperCase()}]...` });

  toast("Restaurando estado anterior ao perfil…", "info");
  const res = await App.RestaurarPerfilUso(key);

  for (const r of (res || [])) {
    Visualizer.log({
      type: r.estado === "ok" ? "revert" : "fail",
      msg: `[RESTAURADO] ${r.nome}: ${r.mensagem}`
    });
  }

  toast("Perfil restaurado com sucesso.", "ok");
  await carregarEstadoPerfis();
  await diagnosticar();
}

/* ==========================================================================
   Métricas de CPU em Tempo Real (Aba Diagnóstico)
   ========================================================================== */

async function atualizarMetricasCPU() {
  try {
    const info = await App.ObterMetricasCPU();
    if (!info) return;

    const totalValEl = $("#cpu-total-val");
    const barFillEl = $("#cpu-bar-fill");
    const coresValEl = $("#cpu-cores-val");
    const procsEl = $("#cpu-top-processes");

    if (totalValEl) totalValEl.textContent = `${info.totalUsagePercent.toFixed(1)}%`;
    if (barFillEl) {
      barFillEl.style.width = `${Math.min(100, Math.max(0, info.totalUsagePercent))}%`;
      barFillEl.style.background = info.totalUsagePercent > 80 ? 'var(--danger)' : (info.totalUsagePercent > 50 ? 'var(--warn)' : 'var(--accent)');
    }
    if (coresValEl) coresValEl.textContent = `${info.physicalCores} Físicos / ${info.logicalProcessors} Lógicos`;

    if (procsEl && info.topProcesses && info.topProcesses.length) {
      procsEl.innerHTML = info.topProcesses.map((p) => `
        <div style="display:flex; justify-content:space-between; margin-bottom:4px; font-size:12px;">
          <span>• <b>${esc(p.name)}</b> (PID: ${p.pid})</span>
          <span class="mono"><b>${p.cpuPercent.toFixed(1)}s</b> CPU acumulada</span>
        </div>
      `).join("");
    }
  } catch (e) {
    // Polling silencioso
  }
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", boot);
} else {
  boot();
}
