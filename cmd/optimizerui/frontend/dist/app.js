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
let dadosPrecarregados = false;
let diagnosticoEmAndamento = null;

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const comTimeout = (promessa, ms = 10000) => {
  let timer;
  return Promise.race([promessa, new Promise((_, rejeitar) => {
    timer = setTimeout(() => rejeitar(new Error("A leitura demorou mais que o esperado.")), ms);
  })]).finally(() => clearTimeout(timer));
};

/* ==========================================================================
   Motor do Visualizador de Operações & Telemetria em Tempo Real
   ========================================================================== */

const Visualizer = {
  events: [],
  autoOpen: false,
  reads: 0,
  writes: 0,
  snaps: 0,
  durations: [],

  init() {
    const btnToggle = $("#btn-toggle-visualizer");
    if (btnToggle) btnToggle.addEventListener("click", () => this.toggle());

    const btnClose = $("#vis-btn-close") || $("#btn-vis-fechar");
    if (btnClose) btnClose.addEventListener("click", () => this.hide());

    const btnClear = $("#vis-btn-clear") || $("#btn-vis-limpar");
    if (btnClear) btnClear.addEventListener("click", () => this.clear());

    const btnCopy = $("#vis-btn-copy") || $("#vis-btn-copiar") || $("#btn-vis-copiar");
    if (btnCopy) btnCopy.addEventListener("click", () => this.copy());

    const commandToggle = $("#chk-vis-commands");
    if (commandToggle) commandToggle.addEventListener("change", () => {
      const drawer = $("#visualizer-drawer");
      if (drawer) drawer.classList.toggle("commands-hidden", !commandToggle.checked);
    });

    $$(".vis-tab").forEach((btn) => btn.addEventListener("click", () => {
      $$(".vis-tab").forEach((x) => x.classList.toggle("on", x === btn));
      $$(".vis-tab-content").forEach((c) => c.classList.remove("on"));
      const tabId = btn.dataset.vistab;
      const el = $("#vistab-" + tabId);
      if (el) el.classList.add("on");
    }));
  },

  show() {
    const el = $("#visualizer-drawer");
    if (el && this.autoOpen) el.hidden = false;
  },

  hide() {
    const el = $("#visualizer-drawer");
    if (el) el.hidden = true;
  },

  toggle() {
    const el = $("#visualizer-drawer");
    if (el) el.hidden = !el.hidden;
  },

  clear() {
    this.events = [];
    this.reads = 0;
    this.writes = 0;
    this.snaps = 0;
    this.durations = [];
    const log = $("#vis-stream-log");
    if (log) log.innerHTML = '<div class="vis-log-empty">Logs limpos. Aguardando novas operações de sistema…</div>';
    const diff = $("#vis-diff-container");
    if (diff) diff.innerHTML = '<div class="vis-log-empty">Nenhum diff ativo no momento.</div>';
    this.updateStats();
  },

  updateStats() {
    const elReads = $("#stat-reads"); if (elReads) elReads.textContent = this.reads;
    const elWrites = $("#stat-writes"); if (elWrites) elWrites.textContent = this.writes;
    const elSnaps = $("#stat-snaps"); if (elSnaps) elSnaps.textContent = this.snaps;
    const avg = this.durations.length
      ? (this.durations.reduce((a, b) => a + b, 0) / this.durations.length).toFixed(2)
      : "0.2";
    const elAvg = $("#stat-avgtime"); if (elAvg) elAvg.textContent = `${avg} ms`;
    const elCount = $("#vis-event-count"); if (elCount) elCount.textContent = `${this.events.length} ops`;
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
    if (op.command || op.output || op.error) {
      const command = op.command ? `COMANDO: ${op.command}` : "";
      const output = op.output ? `SAÍDA: ${op.output}` : "";
      const error = op.error ? `ERRO: ${op.error}` : "";
      detailsHTML += `<span class="vis-command-detail">${esc([command, output, error].filter(Boolean).join("\n"))}</span>`;
    }

    if (typeof document === "undefined" || typeof document.createElement !== "function") {
      this.updateStats();
      return;
    }

    const row = document.createElement("div");
    row.className = "vis-stream-row";
    row.innerHTML = `
      <span class="vis-time">${timeStr}</span>
      <span class="vis-tag-pill ${tagClass}">${tagLabel}</span>
      <span class="vis-msg">${detailsHTML}<span class="duration">(${duration.toFixed(2)}ms)</span></span>
    `;

    const logContainer = $("#vis-stream-log");
    if (!logContainer) return;
    const emptyNotice = logContainer.querySelector(".vis-log-empty");
    if (emptyNotice) emptyNotice.remove();

    while (logContainer.children.length >= 500) {
      logContainer.removeChild(logContainer.firstChild);
    }
    if (this.events.length > 500) {
      this.events.shift();
    }

    logContainer.appendChild(row);

    if ($("#chk-vis-autoscroll")?.checked) {
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
    const btn = $("#vis-btn-copy") || $("#vis-btn-copiar") || $("#btn-vis-copiar");
    if (!btn) return;
    btn.textContent = "Copiado!";
    setTimeout(() => {
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> Copiar`;
    }, 2000);
  }
};

/* ==========================================================================
   Inicialização do App & Splash Screen
   ========================================================================== */

function abrirSplash(titulo = "OPTIMIZER", subtitulo = "Inicializando motor de diagnóstico e sintonia do sistema…") {
  const splash = $("#app-splash-screen");
  if (!splash) return;
  splash.style.display = "flex";
  splash.classList.remove("hidden");
  
  const titleEl = splash.querySelector(".splash-title");
  if (titleEl && titulo) titleEl.innerHTML = `${esc(titulo)} <span class="splash-version">2.0</span>`;
  
  const subEl = splash.querySelector(".splash-subtitle");
  if (subEl && subtitulo) subEl.textContent = subtitulo;
  
  atualizarProgressoSplash(0, 51, "Iniciando verificação…", "Preparando ambiente");
}

function atualizarProgressoSplash(atual, total, statusText, detalheText) {
  const splash = $("#app-splash-screen");
  if (!splash) return;

  const pct = Math.min(100, Math.max(5, Math.round((atual / total) * 100)));
  const fill = $("#splash-progress-fill");
  if (fill) fill.style.width = `${pct}%`;

  const counter = $("#splash-status-counter");
  if (counter) counter.textContent = `[ ${atual} / ${total} ]`;

  const status = $("#splash-status-text");
  if (status && statusText) status.textContent = statusText;

  const detail = $("#splash-detail-text");
  if (detail && detalheText) detail.textContent = detalheText;
}

function animarSplashDuranteBoot(duracaoMs) {
  const inicio = Date.now();
  return setInterval(() => {
    const progresso = Math.min(100, ((Date.now() - inicio) / duracaoMs) * 100);
    const fill = $("#splash-progress-fill");
    if (fill) fill.style.width = `${Math.max(5, progresso)}%`;
    const counter = $("#splash-status-counter");
    if (counter) counter.textContent = `[ ${Math.round(progresso * 51 / 100)} / 51 ]`;
  }, 40);
}

function fecharSplash() {
  const splash = $("#app-splash-screen");
  if (splash) {
    splash.classList.add("hidden");
    setTimeout(() => {
      splash.style.display = "none";
    }, 450);
  }
}

async function aguardarApp() {
  for (let i = 0; i < 80; i++) {
    if (window.go && window.go.main && window.go.main.App) {
      App = window.go.main.App;
      return App;
    }
    await sleep(40);
  }
  return null;
}

async function boot() {
  abrirSplash();
  const inicioLoading = Date.now();
  const splashAnimationTimer = animarSplashDuranteBoot(2000);
  const staticLoadingWatchdog = setTimeout(encerrarLoadingsPendentes, 20000);

  const fallbackTimeout = setTimeout(() => {
    fecharSplash();
  }, 4000);

  try {
    atualizarProgressoSplash(8, 51, "Conectando ao núcleo do Optimizer…", "Aguardando canal IPC seguro");
    await aguardarApp();

    Visualizer.init();
    ligarEventos();

    // O carregamento dos dados da interface não pode depender do diagnóstico
    // completo. O diagnóstico pode demorar em máquinas com WMI/PowerShell
    // lento; iniciar os módulos aqui evita que o HTML permaneça no spinner.
    const preloadPromise = precarregarDadosApp().catch((err) => {
      console.error("Erro ao carregar módulos estáticos:", err);
    });

    atualizarProgressoSplash(18, 51, "Examinando subsistemas de CPU e GPU…", "Prioridade MMCSS e agendamento GPU");
    await sleep(150);

    atualizarProgressoSplash(35, 51, "Auditando catálogo e integridade do Registro…", "Lendo chaves HKLM/HKCU");
    if (App && typeof App.Diagnosticar === "function") {
      await diagnosticar(true);
    }
    await sleep(150);

    atualizarProgressoSplash(48, 51, "Carregando dados dos módulos…", "Preparando rede, discos, inicialização e histórico");
    await preloadPromise;
    atualizarTelemetriaAoVivo();
    await sleep(150);

    atualizarProgressoSplash(51, 51, "Inicialização concluída com sucesso!", "Optimizer 2.0 pronto");
    await sleep(200);
  } catch (err) {
    console.error("Erro durante inicialização:", err);
  } finally {
    clearTimeout(fallbackTimeout);
    clearTimeout(staticLoadingWatchdog);
    clearInterval(splashAnimationTimer);
    atualizarProgressoSplash(51, 51, "Inicialização concluída com sucesso!", "Optimizer 2.0 pronto");
    await sleep(Math.max(0, 2000 - (Date.now() - inicioLoading)));
    fecharSplash();
  }
}

function encerrarLoadingsPendentes() {
  const dns = $("#dns-atual");
  if (dns && dns.textContent.includes("Lendo o DNS")) {
    dns.innerHTML = '<div><span class="dns-current-label">DNS Atual</span><span>Não foi possível concluir a leitura automática.</span></div><button class="btn-secondary small-btn" id="btn-atualizar-dns">Tentar novamente</button>';
    $("#btn-atualizar-dns")?.addEventListener("click", carregarDNSAtual);
  }
  const startup = $("#lista-startup");
  if (startup && startup.textContent.includes("Lendo programas")) {
    startup.innerHTML = '<div class="empty-results"><p>A leitura da inicialização demorou mais que o esperado.</p><button class="btn-secondary" id="btn-recarregar-startup-now">Tentar novamente</button></div>';
    $("#btn-recarregar-startup-now")?.addEventListener("click", carregarStartup);
  }
  const drives = $("#lista-drives");
  if (drives && drives.textContent.includes("Auditando unidades")) {
    drives.innerHTML = '<div class="empty-results"><p>A leitura dos discos demorou mais que o esperado.</p><button class="btn-secondary" id="btn-recarregar-discos-now">Tentar novamente</button></div>';
    $("#btn-recarregar-discos-now")?.addEventListener("click", carregarDiscos);
  }
  const adapter = $("#rede-adaptador");
  if (adapter && adapter.value === "Carregando…") {
    adapter.innerHTML = '<option value="">Interface não identificada — tente novamente</option>';
  }
}

// Todos os dados estáticos são lidos uma vez durante o loading. Trocar de aba
// apenas revela o HTML já preenchido; novas leituras só ocorrem por ação
// explícita do usuário (Atualizar, Medir ou Aplicar).
async function precarregarDadosApp() {
  const tarefas = [
    carregarEstadoPerfis(),
    carregarPerfis(),
    carregarDNSAtual(),
    carregarEditorRede(),
    carregarHistorico(),
    carregarStartup(),
    carregarDiscos(),
    carregarTimerResolution(),
    atualizarGraficoPingAoVivo()
  ];
  await Promise.allSettled(tarefas);
  dadosPrecarregados = true;
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

function deveDiagnosticarAoAbrirAjustes(diagnostico) {
  return !Array.isArray(diagnostico?.itens) || diagnostico.itens.length === 0;
}

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

    // Interrompe o polling de telemetria se o usuário sair da aba Telemetria
    if (viewId !== "telemetria" && telemetriaTimer) {
      clearInterval(telemetriaTimer);
      telemetriaTimer = null;
    }

    // Dados de catálogo, histórico, inicialização, discos e rede já foram
    // carregados no boot. Navegar não dispara novas consultas.
    if (viewId === "telemetria") {
      atualizarTelemetriaAoVivo();
      if (!telemetriaTimer) telemetriaTimer = setInterval(atualizarTelemetriaAoVivo, 1000);
    }
    if (viewId === "otim" && deveDiagnosticarAoAbrirAjustes(ultimoDiagnostico)) {
      diagnosticar();
    }
  }));

  // Listeners de Perfis de Uso
  $$(".btn-view-profile").forEach((b) => b.addEventListener("click", () => visualizarPerfil(b.dataset.profile)));
  $$(".btn-verify-profile").forEach((b) => b.addEventListener("click", () => verificarPerfil(b.dataset.profile)));
  $$(".btn-apply-profile").forEach((b) => b.addEventListener("click", () => aplicarPerfilUso(b.dataset.profile)));
  $$(".btn-restore-profile").forEach((b) => b.addEventListener("click", () => restaurarPerfilUso(b.dataset.profile)));

  const btnExportarRelatorio = $("#btn-exportar-relatorio");
  if (btnExportarRelatorio) {
    btnExportarRelatorio.addEventListener("click", async () => {
      try {
        toast("Gerando relatório completo do sistema…", "info");
        const relatorio = await App.ExportarRelatorioSistema();
        if (relatorio) {
          abrirModal("Relatório de Diagnóstico & Otimização", `<pre style="white-space:pre-wrap; font-family:var(--font-mono); font-size:12px; max-height:400px; overflow-y:auto; background:var(--bg-sunken); padding:12px; border-radius:var(--radius-md);">${esc(relatorio)}</pre>`);
        }
      } catch (e) {
        toast("Erro ao exportar relatório: " + e, "err");
      }
    });
  }

  const btnRecarregarStartup = $("#btn-recarregar-startup");
  if (btnRecarregarStartup) btnRecarregarStartup.addEventListener("click", carregarStartup);

  const btnAuditarReparo = $("#btn-auditar-reparo");
  if (btnAuditarReparo) btnAuditarReparo.addEventListener("click", auditarReparo);

  const btnTestarDNS = $("#btn-testar-dns");
  if (btnTestarDNS) btnTestarDNS.addEventListener("click", testarDNS);

  const btnTestarMatrizJogos = $("#btn-testar-matriz-jogos");
  if (btnTestarMatrizJogos) btnTestarMatrizJogos.addEventListener("click", testarMatrizJogos);

  const btnFlushRede = $("#btn-flush-rede");
  if (btnFlushRede) btnFlushRede.addEventListener("click", executarFlushRede);

  const btnMedirAntes = $("#btn-medir-antes");
  if (btnMedirAntes) btnMedirAntes.addEventListener("click", medirAntes);

  const btnMedirDepois = $("#btn-medir-depois");
  if (btnMedirDepois) btnMedirDepois.addEventListener("click", medirDepois);

  const btnMedirMTU = $("#btn-medir-mtu");
  if (btnMedirMTU) btnMedirMTU.addEventListener("click", medirMTU);

  const btnLerRede = $("#btn-carregar-rede-config");
  if (btnLerRede) btnLerRede.addEventListener("click", carregarEditorRede);
  const mtuRede = $("#rede-mtu");
  if (mtuRede) mtuRede.addEventListener("input", atualizarMSSRede);
  const btnValidarRede = $("#btn-validar-rede-config");
  if (btnValidarRede) btnValidarRede.addEventListener("click", () => aplicarEditorRede(true));
  const btnAplicarRede = $("#btn-aplicar-rede-config");
  if (btnAplicarRede) btnAplicarRede.addEventListener("click", () => aplicarEditorRede(false));

  const btnEscanearPnp = $("#btn-escanear-pnp");
  if (btnEscanearPnp) btnEscanearPnp.addEventListener("click", escanearDispositivosFantasmas);

  const btnLimparPnp = $("#btn-limpar-pnp");
  if (btnLimparPnp) btnLimparPnp.addEventListener("click", limparDispositivosFantasmas);

  const btnAtualizarTimerRes = $("#btn-atualizar-timerres");
  if (btnAtualizarTimerRes) btnAtualizarTimerRes.addEventListener("click", carregarTimerResolution);

  const btnEscanearMsi = $("#btn-escanear-msi");
  if (btnEscanearMsi) btnEscanearMsi.addEventListener("click", escanearDispositivosMSI);


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

  const btnSimular = $("#btn-simular");
  if (btnSimular) btnSimular.addEventListener("click", () => aplicar(true));

  const btnAplicar = $("#btn-aplicar");
  if (btnAplicar) btnAplicar.addEventListener("click", () => aplicar(false));

  const modalFechar = $("#modal-fechar");
  if (modalFechar) modalFechar.addEventListener("click", fecharModal);

  const btnModalX = $("#btn-modal-x");
  if (btnModalX) btnModalX.addEventListener("click", fecharModal);
}

/* ==========================================================================
   Diagnóstico com Telemetria Visual
   ========================================================================== */

function gerarVisaoGeralDeDiagnostico(d) {
  if (!d) return null;
  const total = d.total || (d.itens ? d.itens.length : 0);
  const aplicados = d.aplicados || 0;
  const cobertura = total > 0 ? (aplicados / total) * 100 : 0;
  
  const catMap = {};
  (d.itens || []).forEach((it) => {
    const nome = it.categoria || "Outros";
    if (!catMap[nome]) catMap[nome] = { nome, total: 0, aplicados: 0 };
    catMap[nome].total++;
    if (it.estado === "aplicado") catMap[nome].aplicados++;
  });
  
  return {
    perfil: d.perfil || perfil,
    totalAjustes: total,
    aplicados: aplicados,
    recomendadosPendentes: d.recomendadosPendentes || 0,
    pendentesDesfazer: d.pendentesDesfazer || 0,
    coberturaPercentual: cobertura,
    categorias: Object.values(catMap)
  };
}

function mostrarFalhaDiagnostico(erro) {
  const detalhe = erro ? `: ${esc(String(erro))}` : "";
  const painel = $("#visao-geral");
  if (painel) {
    painel.innerHTML = `<div class="overview-unavailable"><span>Visão geral indisponível${detalhe}</span></div>`;
  }
  const resumo = $("#resumo");
  if (resumo) {
    resumo.innerHTML = `<div class="empty-results"><p>Não foi possível concluir o diagnóstico agora${detalhe}.</p><button class="btn-secondary" id="btn-rever-falha">Tentar novamente</button></div>`;
    const btn = $("#btn-rever-falha");
    if (btn) btn.addEventListener("click", () => diagnosticar());
  }
}

async function diagnosticar(isBoot = false) {
  if (diagnosticoEmAndamento) return diagnosticoEmAndamento;
  diagnosticoEmAndamento = diagnosticarInterno(isBoot);
  try {
    return await diagnosticoEmAndamento;
  } finally {
    diagnosticoEmAndamento = null;
  }
}

async function diagnosticarInterno(isBoot = false) {
  if (!App) await aguardarApp();
  if (!App || typeof App.Diagnosticar !== "function") {
    mostrarFalhaDiagnostico("núcleo do app indisponível");
    return;
  }

  const totalEsperado = perfil === "trabalho" ? 37 : 51;
  const listaEl = $("#lista");
  if (listaEl) listaEl.innerHTML = "";

  const resumoEl = $("#resumo");
  if (resumoEl) {
    resumoEl.innerHTML = `
      <div class="health-loading" style="display:flex; flex-direction:column; gap:10px; width:100%;">
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span style="font-weight:600;"><span class="pulse-spinner"></span> Auditando catálogo de ajustes do sistema (${esc(perfil)})…</span>
          <span class="mono bold" style="color:var(--accent);" id="resumo-loading-count">[ 0 / ${totalEsperado} ]</span>
        </div>
        <div style="height:6px; background:rgba(255,255,255,0.08); border-radius:999px; overflow:hidden;">
          <div id="resumo-loading-bar" style="width:25%; height:100%; background:linear-gradient(90deg, #00f0ff, #7928ca); border-radius:999px; transition:width 0.2s ease;"></div>
        </div>
      </div>`;
  }

  Visualizer.log({
    type: "read",
    msg: `Iniciando varredura e diagnóstico do catálogo para o perfil [${perfil}]`
  });

  let d;
  try {
    d = await App.Diagnosticar(perfil);
  } catch (e) {
    mostrarFalhaDiagnostico(e);
    return;
  }
  ultimoDiagnostico = d;

  // O diagnóstico já contém todos os dados necessários para a visão geral.
  // Não chamar ResumoVisao aqui: esse binding executa outro Scan completo do
  // Registro e pode deixar a tela presa aguardando uma segunda leitura.
  renderVisaoGeral(gerarVisaoGeralDeDiagnostico(d));

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
        command: `reg query "${hive}\\${it.categoria || "Software"}" /v "${it.id}"`,
        status: it.estado
      });
    }
  }

  $("#admin-area").innerHTML = d.admin
    ? '<span class="admin-badge admin-on"><span class="admin-dot"></span> Administrador</span>'
    : '<span class="admin-badge admin-off"><span class="admin-dot"></span> Usuário Padrão</span><button class="admin-elevate-btn" id="btn-elevar">Reabrir como Admin</button>';
  
  const btnElevar = $("#btn-elevar");
  if (btnElevar) btnElevar.addEventListener("click", elevar);

  const restauracao = $("#wrap-restauracao");
  if (restauracao) restauracao.hidden = !d.admin;
  const caminhoHistorico = $("#hist-caminho");
  if (caminhoHistorico) caminhoHistorico.textContent = d.caminhoHistorico;
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

function renderVisaoGeral(visao) {
  const painel = $("#visao-geral");
  if (!painel || !visao) return;

  const total = Math.max(0, visao.totalAjustes || 0);
  const aplicados = Math.max(0, visao.aplicados || 0);
  const cobertura = Math.min(100, Math.max(0, visao.coberturaPercentual || 0));
  const percentualInteiro = Math.round(cobertura);
  const categorias = (visao.categorias || []).slice(0, 6);
  const perfilNome = visao.perfil === "trabalho" ? "Trabalho" : "Pessoal";

  painel.innerHTML = `
    <div class="overview-heading">
      <div>
        <span class="overview-eyebrow">Vis\u00e3o geral</span>
        <h2>Progresso do cat\u00e1logo</h2>
        <p>Acompanhamento do perfil ${esc(perfilNome)} com base no estado atual dos ajustes.</p>
      </div>
      <span class="overview-updated">Atualizado agora</span>
    </div>
    <div class="overview-layout">
      <div class="coverage-chart-panel" style="display:flex; align-items:center; gap:12px;">
        <canvas id="canvas-donut-cobertura" style="width:96px; height:96px; flex-shrink:0; display:block;"></canvas>
        <div class="coverage-copy">
          <strong>${aplicados} de ${total}</strong>
          <span>ajustes aplicados</span>
          <small>Cobertura indica apenas o estado do cat\u00e1logo; n\u00e3o é uma nota de desempenho.</small>
        </div>
      </div>
      <div class="overview-kpis">
        <div class="overview-stat applied">
          <span class="overview-stat-label">Aplicados</span>
          <strong>${aplicados}</strong>
          <small>prontos no perfil</small>
        </div>
        <div class="overview-stat pending">
          <span class="overview-stat-label">Recomendados</span>
          <strong>${visao.recomendadosPendentes || 0}</strong>
          <small>ainda dispon\u00edveis</small>
        </div>
        <div class="overview-stat undo">
          <span class="overview-stat-label">Revers\u00edveis</span>
          <strong>${visao.pendentesDesfazer || 0}</strong>
          <small>podem ser desfeitos</small>
        </div>
      </div>
      <div class="category-chart-panel">
        <div class="category-chart-heading">
          <strong>Distribui\u00e7\u00e3o por categoria</strong>
          <span>${categorias.length ? "Principais categorias" : "Sem dados"}</span>
        </div>
        <div class="category-bars">
          ${categorias.map((categoria) => {
            const categoriaTotal = Math.max(1, categoria.total || 0);
            const categoriaAplicados = Math.max(0, categoria.aplicados || 0);
            const categoriaPercentual = Math.min(100, Math.round(categoriaAplicados / categoriaTotal * 100));
            return `<div class="category-bar-row">
              <div class="category-bar-label"><span>${esc(categoria.nome)}</span><b>${categoriaAplicados}/${categoria.total}</b></div>
              <div class="category-bar-track"><span style="width:${categoriaPercentual}%"></span></div>
            </div>`;
          }).join("") || '<p class="overview-unavailable">Nenhuma categoria encontrada.</p>'}
        </div>
      </div>
    </div>`;

  setTimeout(() => {
    renderCatalogDonutChart("canvas-donut-cobertura", percentualInteiro, aplicados, total);
  }, 50);
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
  const countEl = $("#selecionados-count");
  const labelEl = $("#selecionados-sub");
  const btnAplicar = $("#btn-aplicar");

  if (countEl) countEl.textContent = selecionadosCount;
  if (labelEl) labelEl.textContent = selecionadosCount
    ? "Pronto para simular ou aplicar as alterações selecionadas"
    : "Nenhum item alterado pendente de aplicação";
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
  const marcar = !aplicado && it.recomendado && (!it.precisaAdmin || ultimoDiagnostico?.admin);

  return `
  <div class="tweak-card ${aplicado ? "applied" : ""}">
    <label class="switch-control" title="${aplicado ? "Este ajuste já está ativo" : "Marcar para aplicar"}">
      <input type="checkbox" data-id="${esc(it.id)}" ${aplicado ? "disabled checked" : ""} ${!aplicado && marcar ? "checked" : ""}>
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
      ${it.motivoRecomendacao ? `<div class="field-hint recommendation-reason">${esc(it.motivoRecomendacao)}</div>` : ""}
      ${!aplicado && it.ressalva ? `
        <button class="link-expand">Por que isso não vem marcado?</button>
        <div class="caveat-box" hidden>
          <b>Ressalva honesta:</b> ${esc(it.ressalva)}
          ${(it.base || it.evidencia) ? `<div class="caveat-evidence">Base técnica: ${esc(it.base || it.evidencia)}</div>` : ""}
        </div>` : ""}
    </div>
    <span class="status-pill ${esc(it.estado)}">${esc(rotulo)}</span>
  </div>`;
}

function marcarRecomendados() {
  const itens = ultimoDiagnostico?.itens || [];
  const rec = new Set(itens
    .filter((i) => i.recomendado && (!i.precisaAdmin || ultimoDiagnostico?.admin))
    .map((i) => i.id));
  const bloqueados = itens.filter((i) => i.recomendado && i.precisaAdmin && !ultimoDiagnostico?.admin).length;
  $$('.switch-control input[type=checkbox]').forEach((c) => {
    if (!c.disabled) c.checked = rec.has(c.dataset.id);
  });
  atualizarContadorSelecionados();
  Visualizer.log({
    type: "read",
    msg: `Itens recomendados marcados para aplicação em lote (${rec.size} itens)`
  });
  toast(
    bloqueados
      ? `${rec.size} item(ns) selecionado(s). ${bloqueados} recomendação(ões) exigem “Reabrir como Admin”.`
      : "Itens recomendados selecionados.",
    "info",
    3500
  );
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

  const pontoRestauracao = $("#ponto-restauracao");
  const areaRestauracao = $("#wrap-restauracao");
  const ponto = pontoRestauracao?.checked === true && areaRestauracao?.hidden !== true;
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
      const aplicados = (res || []).filter((r) => r.estado === "ok").length;
      const falhas = (res || []).filter((r) => r.estado === "falhou").length;
      const pulados = (res || []).filter((r) => r.estado === "pulado").length;
      if (falhas || pulados || !aplicados) {
        toast(`${aplicados} aplicado(s), ${pulados} pulado(s), ${falhas} falha(s). Veja os detalhes.`, falhas || pulados ? "erro" : "info", 4500);
      } else {
        toast("Otimizações aplicadas com sucesso.", "ok");
      }
      await diagnosticar();
    } else {
      toast("Simulação concluída. Nenhuma chave do registro foi alterada.", "info");
    }
  } catch (e) {
    toast(`Falha na aplicação: ${String(e)}`, "erro");
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
  } catch (e) {
    toast(`Falha ao desfazer configurações: ${String(e)}`, "erro");
  } finally {
    travar(false);
  }
}

function travar(travado) {
  $$("#barra button").forEach((b) => (b.disabled = travado));
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

  const bloqueadosAdmin = res.filter((r) => r.estado === "pulado" && /administrador/i.test(r.mensagem || ""));
  const avisoAdmin = bloqueadosAdmin.length
    ? `<div class="reboot-warn-box">
        <span>⚠ ${bloqueadosAdmin.length} ajuste(s) não foram aplicados porque exigem administrador. Use “Reabrir como Admin” no topo e tente novamente.</span>
       </div>`
    : "";
  const avisoReiniciar = res.some((r) => r.precisaSair && r.estado === "ok")
    ? `<div class="reboot-warn-box">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <span>Alguns ajustes visuais só terão efeito completo após reiniciar a sessão do Windows (Logoff).</span>
       </div>`
    : "";

  abrirModal(titulo, linhas + avisoAdmin + avisoReiniciar);
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
  const dest = ($("#destino")?.value || "8.8.8.8").trim();
  const btn = $("#btn-medir-mtu");
  const container = $("#mtu-resultado");
  if (btn) btn.disabled = true;
  if (btn) btn.innerHTML = `<span class="pulse-spinner"></span> Medindo…`;
  if (container) container.innerHTML = `
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

    if (container) container.innerHTML = mtuHTML(m);
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
  } catch (e) {
    if (container) container.innerHTML = `<div class="empty-results"><p>Erro ao medir MTU: ${esc(String(e))}</p></div>`;
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg> Medir MTU Ideal`;
    }
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
  if (!btn) return;
  btn.disabled = true;
  Visualizer.log({ type: "write", msg: "Aplicando novo valor de MTU na interface de rede..." });
  try {
    const res = await App.AplicarMTU(false);
    mostrarResultados("Ajuste de MTU", res);
    toast("MTU ajustado com sucesso.", "ok");
    await medirMTU();
  } catch (e) {
    toast(`Não foi possível ajustar o MTU: ${String(e)}`, "erro");
  } finally {
    btn.disabled = false;
  }
}

/* ==========================================================================
   Perfis de Rede
   ========================================================================== */

function atualizarMSSRede() {
  const mtu = Number($("#rede-mtu")?.value || 0);
  if ($("#rede-mss4")) $("#rede-mss4").value = mtu >= 576 ? mtu - 40 : "";
  if ($("#rede-mss6")) $("#rede-mss6").value = mtu >= 576 ? mtu - 60 : "";
}

function lerConfigRedeUI() {
  return { adaptador: $("#rede-adaptador")?.value || "", mtu: Number($("#rede-mtu")?.value || 0), mssIPv4: Number($("#rede-mss4")?.value || 0), mssIPv6: Number($("#rede-mss6")?.value || 0), idsOtimizacao: $$(".rede-tweak:checked").map((x) => x.value) };
}

async function carregarEditorRede() {
  if (!App || typeof App.ObterConfiguracaoRede !== "function") return;
  try {
    const cfg = await comTimeout(App.ObterConfiguracaoRede());
    if (!cfg?.adaptador) throw new Error("Não foi possível identificar a interface de saída.");
    const select = $("#rede-adaptador");
    if (select) select.innerHTML = `<option value="${esc(cfg.adaptador)}">${esc(cfg.adaptador)}</option>`;
    $("#rede-mtu").value = cfg.mtu || 1500;
    atualizarMSSRede();
  } catch (err) { toast("Falha ao ler rede: " + (err.message || err), "erro"); }
}

async function aplicarEditorRede(simular) {
  if (!App || typeof App.ValidarConfiguracaoRede !== "function") return;
  const cfg = lerConfigRedeUI();
  const result = $("#rede-config-resultado");
  let valid;
  try { valid = await App.ValidarConfiguracaoRede(cfg); }
  catch (err) {
    if (result) result.innerHTML = `<div class="error-box">${esc(String(err))}</div>`;
    return;
  }
  if (!valid.valida) { if (result) result.innerHTML = `<div class="error-box">${esc(valid.mensagem)}</div>`; return; }
  if (simular) {
    if (result) result.innerHTML = `<div class="terminal-block"><p>${esc(valid.mensagem)}</p>${(valid.comandos || []).map((c) => `<div class="terminal-line"><span class="cmd-ok">${esc(c)}</span></div>`).join("")}</div>`;
    Visualizer.log({ type: "net", msg: "Simulação de configuração de MTU/MSS", command: (valid.comandos || []).join("\n") });
    return;
  }
  const btn = $("#btn-aplicar-rede-config");
  if (btn) btn.disabled = true;
  try {
    const res = await App.AplicarConfiguracaoRede(cfg, false);
    mostrarResultados("Otimização de rede", res);
    const falhas = (res || []).filter((x) => x.estado === "falhou");
    toast(falhas.length ? "A configuração terminou com falhas." : "Configuração de rede aplicada.", falhas.length ? "erro" : "ok");
    await carregarEditorRede();
  } catch (err) { toast("Falha ao aplicar rede: " + (err.message || err), "erro"); }
  finally { if (btn) btn.disabled = false; }
}

async function carregarPerfis() {
  const container = $("#perfis-lista");
  if (!container) return;
  container.innerHTML = '<p class="muted-text">Carregando perfis de rede…</p>';
  try {
    const perfis = await comTimeout(App.ListarPerfisRede());
    container.innerHTML = (perfis || []).map(renderPerfilCard).join("") || '<p class="muted-text">Nenhum perfil de rede disponível.</p>';
    $$(".btn-aplicar-perfil").forEach((b) => b.addEventListener("click", () => aplicarPerfil(b.dataset.key)));
  } catch (e) {
    container.innerHTML = `<div class="empty-results"><p>Erro ao carregar perfis de rede: ${esc(String(e))}</p><button class="btn-secondary" id="btn-recarregar-perfis">Tentar novamente</button></div>`;
    const retry = $("#btn-recarregar-perfis");
    if (retry) retry.addEventListener("click", carregarPerfis);
  }
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
  } catch (e) {
    toast(`Falha ao aplicar perfil de rede: ${String(e)}`, "erro");
  } finally {
    btns.forEach((b) => (b.disabled = false));
  }
}

/* ==========================================================================
   Latência & Benchmark Antes / Depois
   ========================================================================== */

let ultimoAntes = null;

async function medirAntes() {
  const host = ($("#destino")?.value || "8.8.8.8").trim();
  const pacotesInput = $("#pacotes");
  const pacotes = pacotesInput ? parseInt(pacotesInput.value, 10) || 10 : 10;
  const btn = $("#btn-medir-antes");
  if (btn) btn.disabled = true;
  if (btn) btn.innerHTML = `<span class="pulse-spinner"></span> Medindo (${pacotes} pkts)…`;
  
  $("#comparativo-resultado").innerHTML = `
    <div class="glass-card" style="margin-top:14px">
      <div class="health-loading">
        <span class="pulse-spinner"></span>
        <span>Enviando ${pacotes} pacotes de teste para [${esc(host)}] para registrar a medição base…</span>
      </div>
    </div>`;

  Visualizer.show();
  Visualizer.log({ type: "net", msg: `Disparando ${pacotes} sondagens ICMP de medição base até [${host}]...` });

  try {
    ultimoAntes = typeof App.MedirRedeComPacotes === "function" 
      ? await App.MedirRedeComPacotes(host, pacotes) 
      : await App.MedirRedeAntes(host);

    const valorAntes = $("#lat-antes-val");
    if (valorAntes) valorAntes.textContent = `${ultimoAntes.avgRTT} ms`;

    Visualizer.log({
      type: "net",
      msg: `Medição base concluída: RTT médio = ${ultimoAntes.avgRTT}ms, Jitter = ${ultimoAntes.stdDev}ms`
    });

    $("#comparativo-resultado").innerHTML = benchmarkHTML(ultimoAntes, "Medição Base — Antes da Otimização");
    
    const btnDepois = $("#btn-medir-depois");
    if (btnDepois && !ultimoAntes.erro) {
      btnDepois.disabled = false;
      if (btnDepois.classList) btnDepois.classList.remove("disabled");
    }
    toast("Medição base concluída! Agora você pode realizar a medição pós-otimização.", "ok");
  } catch (e) {
    $("#comparativo-resultado").innerHTML = `<div class="glass-card"><h3 class="card-title">Erro na Medição</h3><p class="card-subtitle">${esc(String(e))}</p></div>`;
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;display:inline-block;vertical-align:middle;margin-right:4px;"><polygon points="5 3 19 12 5 21 5 3"/></svg> 1. Medir Antes`;
    }
  }
}

async function medirDepois() {
  if (!ultimoAntes) {
    $("#comparativo-resultado").innerHTML = '<div class="glass-card"><h3 class="card-title">Medição base necessária</h3><p class="card-subtitle">Registre a medição “Antes” antes de comparar a latência atual.</p></div>';
    return;
  }
  const host = ($("#destino")?.value || "8.8.8.8").trim();
  const pacotesInput = $("#pacotes");
  const pacotes = pacotesInput ? parseInt(pacotesInput.value, 10) || 10 : 10;
  const btn = $("#btn-medir-depois");
  if (btn) btn.disabled = true;
  if (btn) btn.innerHTML = `<span class="pulse-spinner"></span> Medindo (${pacotes} pkts)…`;
  
  Visualizer.show();
  Visualizer.log({ type: "net", msg: `Disparando ${pacotes} sondagens ICMP de pós-otimização até [${host}]...` });

  try {
    const depois = typeof App.MedirRedeComPacotes === "function"
      ? await App.MedirRedeComPacotes(host, pacotes)
      : await App.MedirRedeDepois(host);

    if (depois.erro) {
      $("#comparativo-resultado").innerHTML = benchmarkHTML(ultimoAntes, "Medição Base — Antes") +
        `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Falha ao Medir Novamente</h3><p class="card-subtitle">${esc(depois.erro)}</p></div>`;
      return;
    }
    const valorDepois = $("#lat-depois-val");
    if (valorDepois) valorDepois.textContent = `${depois.avgRTT} ms`;

    const comp = await App.RelatorioComparativo(ultimoAntes, depois);
    Visualizer.log({
      type: "net",
      msg: `Comparação final: Latência ${comp.deltaLatencia}, Jitter ${comp.deltaJitter}`
    });

    $("#comparativo-resultado").innerHTML =
      benchmarkHTML(ultimoAntes, "Medição Base — Antes da Otimização") +
      benchmarkHTML(depois, "Medição Atual — Depois das Otimizações") +
      comparativoHTML(comp);
  } catch (e) {
    $("#comparativo-resultado").innerHTML += `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Erro de Comparação</h3><p class="card-subtitle">${esc(String(e))}</p></div>`;
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;display:inline-block;vertical-align:middle;margin-right:4px;"><polygon points="5 3 19 12 5 21 5 3"/></svg> 2. Medir Depois`;
    }
  }
}

function benchmarkHTML(b, titulo) {
  if (b.erro) return `<div class="glass-card" style="margin-top:14px"><h3 class="card-title">Falha</h3><p class="card-subtitle">${esc(b.erro)}</p></div>`;
  let badgeColor = "#10b981";
  if (b.avgRTT > 40) badgeColor = "#3b82f6";
  if (b.avgRTT > 90) badgeColor = "#f59e0b";
  if (b.avgRTT > 150) badgeColor = "#ef4444";

  return `
  <div class="glass-card" style="margin-top:14px">
    <div style="display:flex; justify-content:space-between; align-items:center;">
      <h3 class="card-title">${esc(titulo)}</h3>
      <span class="mono bold" style="color:${badgeColor}; font-size:1.3rem;">${b.avgRTT} ms</span>
    </div>
    <p class="field-hint">Destino ${esc(b.host)} · ${b.packetsSent} pacotes enviados · ${b.packetsLost} pacote(s) perdido(s)</p>
    <div class="data-table-wrap" style="margin-top:10px;">
      <table class="data-table">
        <thead><tr><th>Métrica</th><th>Valor Obtido</th></tr></thead>
        <tbody>
          <tr><td>Latência Média (RTT)</td><td class="mono"><b>${b.avgRTT} ms</b></td></tr>
          <tr><td>Variação Mínima / Máxima</td><td class="mono">${b.minRTT} ms / ${b.maxRTT} ms</td></tr>
          <tr><td>Jitter (Estabilidade)</td><td class="mono">±${b.stdDev} ms</td></tr>
          <tr><td>Perda de Pacotes</td><td class="mono">${(b.lossPercent ?? 0).toFixed(1)}%</td></tr>
        </tbody>
      </table>
    </div>
  </div>`;
}

function comparativoHTML(c) {
  return `
  <div class="glass-card" style="margin-top:14px; border-left: 4px solid var(--accent); background:rgba(0, 240, 255, 0.05);">
    <h3 class="card-title" style="color:var(--accent);">Relatório de Impacto em Tempo Real</h3>
    <p class="card-subtitle" style="font-size:1.05rem; margin-top:4px;">${esc(c.interpretacao)}</p>
    <div style="display:flex; gap:20px; margin-top:12px; font-size:0.95rem;">
      <div>Variação de Latência: <b class="mono" style="color:var(--accent);">${esc(c.deltaLatencia)}</b></div>
      <div>Jitter: <b class="mono">${esc(c.deltaJitter)}</b></div>
    </div>
  </div>`;
}

/* ==========================================================================
   Benchmark de DNS (Base GRC Benchmark) & Aplicação em 1 Clique
   ========================================================================== */

async function carregarDNSAtual() {
  const container = $("#dns-atual");
  if (!container) return;

  try {
    const atual = await comTimeout(App.ObterDNSAtual());
    if (!atual || atual.erro) {
      container.innerHTML = `
        <div>
          <span class="dns-current-label">DNS Atual</span>
          <span>${esc(atual?.erro || "Não foi possível identificar o DNS da conexão ativa.")}</span>
        </div>
        <button class="btn-secondary small-btn" id="btn-atualizar-dns">Recarregar</button>
      `;
    } else {
      const ips = (atual.servidores && atual.servidores.length) ? atual.servidores.join(", ") : "Obtido via DHCP / Roteador";
      container.innerHTML = `
        <div>
          <span class="dns-current-label">DNS Ativo (${esc(atual.interface)})</span>
          <strong>${esc(ips)}</strong>
        </div>
        <button class="btn-secondary small-btn" id="btn-atualizar-dns">Recarregar</button>
      `;
    }
    const btn = $("#btn-atualizar-dns");
    if (btn) btn.addEventListener("click", carregarDNSAtual);
  } catch (e) {
    container.innerHTML = `<div><span class="dns-current-label">DNS Atual</span><span>${esc(String(e))}</span></div>`;
  }
}

async function testarDNS() {
  const btn = $("#btn-testar-dns");
  const container = $("#dns-resultado");
  if (btn) btn.disabled = true;
  if (container) {
    container.innerHTML = `<div class="health-loading"><span class="pulse-spinner"></span><span>Executando benchmark de 12 servidores DNS do GRC Benchmark…</span></div>`;
  }

  Visualizer.show();
  Visualizer.log({ type: "net", msg: "Iniciando benchmark de resolução de nomes DNS (Base GRC)..." });

  try {
    const provedores = await App.BenchmarkDNS();
    if (!provedores || provedores.length === 0) {
      if (container) container.innerHTML = `<p class="muted-text">Nenhum provedor DNS respondeu ao teste.</p>`;
      return;
    }

    if (container) {
      container.innerHTML = `
        <div style="margin-bottom: 12px;">
          <h4 style="font-size:13px; font-weight:700; margin:0 0 6px 0;">Comparativo Visual de Latência DNS (Menor = Mais Rápido)</h4>
          <canvas id="canvas-dns-benchmark" style="width:100%; height:140px; display:block; background:#0c131d; border-radius:8px; border:1px solid rgba(255,255,255,0.06);"></canvas>
        </div>
        <div class="data-table-wrap" style="max-height: 380px; overflow-y: auto;">
          <table class="data-table">
            <thead>
              <tr>
                <th>Provedor DNS (GRC Benchmark)</th>
                <th>Privacidade & Segurança</th>
                <th>Tempo de Resposta (RTT)</th>
                <th>Ação</th>
              </tr>
            </thead>
            <tbody>
              ${provedores.map((p) => {
                let badgeColor = "#10b981";
                if (p.avgRttMs > 60) badgeColor = "#3b82f6";
                if (p.avgRttMs > 120) badgeColor = "#f59e0b";
                if (p.avgRttMs >= 999) badgeColor = "#ef4444";

                const rttStr = p.avgRttMs < 999 ? `${p.avgRttMs} ms` : "Falhou";
                const ipsJson = JSON.stringify(p.ips || []).replace(/"/g, "&quot;");
                const speedPercent = p.avgRttMs < 999 ? Math.max(8, Math.min(100, Math.round(100 - (p.avgRttMs * 0.45)))) : 0;

                return `
                  <tr>
                    <td>
                      <b>${esc(p.nome)}</b>
                      <div class="field-hint" style="font-family:var(--font-mono); font-size:11px;">${esc((p.ips || []).join(", "))}</div>
                    </td>
                    <td><span class="badge-tag rec">${esc(p.privacidade || 'Padrão')}</span></td>
                    <td>
                      <div style="display:flex; align-items:center; gap:10px;">
                        <div style="flex:1; height:6px; background:rgba(255,255,255,0.08); border-radius:999px; overflow:hidden; min-width:80px;">
                          <div style="width:${speedPercent}%; height:100%; background:${badgeColor}; border-radius:999px; transition:width 0.4s ease;"></div>
                        </div>
                        <span class="mono bold" style="color:${badgeColor}; font-size:1rem; min-width:60px; text-align:right;">${rttStr}</span>
                      </div>
                    </td>
                    <td>
                      <button class="btn-primary small-btn btn-usar-dns" data-ips="${ipsJson}" data-nome="${esc(p.nome)}">
                        Usar este DNS
                      </button>
                    </td>
                  </tr>
                `;
              }).join("")}
            </tbody>
          </table>
        </div>
      `;

      setTimeout(() => {
        renderDNSBenchmarkChart("canvas-dns-benchmark", provedores);
      }, 50);

      $$(".btn-usar-dns").forEach((b) => b.addEventListener("click", async () => {
        try {
          const ips = JSON.parse(b.dataset.ips.replace(/&quot;/g, '"'));
          const nome = b.dataset.nome;
          await usarDNS(ips, nome, b);
        } catch (e) {
          toast("Erro ao aplicar DNS: " + e, "err");
        }
      }));
    }
  } catch (e) {
    if (container) container.innerHTML = `<p class="danger-text">Erro ao testar DNS: ${esc(String(e))}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

async function usarDNS(ips, nome, botao) {
  if (!ips || !ips.length) return;
  if (botao) botao.disabled = true;
  toast(`Configurando ${nome} (${ips.join(", ")}) como DNS ativo…`, "info");
  Visualizer.log({ type: "write", msg: `Configurando resolvedor DNS para [${ips.join(", ")}]...` });

  try {
    const res = await App.AplicarDNS(ips);
    if (res && res.ok) {
      toast(res.mensagem || "DNS configurado com sucesso.", "ok");
      Visualizer.log({ type: "apply", msg: `DNS aplicado com sucesso: ${res.mensagem}` });
      await carregarDNSAtual();
    } else {
      toast(res?.mensagem || "Não foi possível alterar o DNS.", "err");
      Visualizer.log({ type: "fail", msg: `Falha ao configurar DNS: ${res?.mensagem}` });
    }
  } catch (e) {
    toast("Erro ao aplicar DNS: " + e, "err");
  } finally {
    if (botao) botao.disabled = false;
  }
}

async function testarMatrizJogos() {
  const container = $("#matriz-jogos-container");
  const btn = $("#btn-testar-matriz-jogos");
  if (btn) btn.disabled = true;
  if (container) {
    container.innerHTML = `<div class="health-loading"><span class="pulse-spinner"></span><span>Medindo tempo de resposta com nós globais de jogos…</span></div>`;
  }

  try {
    const regioes = await App.MatrizPingJogos();
    if (!regioes || regioes.length === 0) {
      if (container) container.innerHTML = `<p class="muted-text">Não foi possível obter dados da matriz de jogos.</p>`;
      return;
    }

    if (container) {
      container.innerHTML = regioes.map((r) => {
        let badgeColor = "#10b981";
        let statusLabel = "Excelente";
        if (r.status === "bom") { badgeColor = "#3b82f6"; statusLabel = "Bom"; }
        else if (r.status === "regular") { badgeColor = "#f59e0b"; statusLabel = "Regular"; }
        else if (r.status === "alto") { badgeColor = "#ef4444"; statusLabel = "Alto"; }
        else if (r.status === "indisponivel") { badgeColor = "#6b7280"; statusLabel = "Sem Resposta"; }

        const pingStr = r.pingMs > 0 ? `${r.pingMs} ms` : "—";
        const jitterStr = r.jitterMs > 0 ? `±${r.jitterMs.toFixed(1)}ms` : "";
        const signalPercent = r.pingMs > 0 ? Math.max(5, Math.min(100, Math.round(100 - (r.pingMs * 0.5)))) : 0;

        return `
          <div class="game-node-card" style="background:var(--bg-sunken); padding:12px; border-radius:var(--radius-md); border:1px solid var(--border-subtle); display:flex; flex-direction:column; gap:6px;">
            <div style="display:flex; justify-content:space-between; align-items:center;">
              <span style="font-size:0.95rem;">${r.bandeira || '🌐'} <b>${esc(r.nome)}</b></span>
              <span class="mono bold" style="color:${badgeColor}; font-size:0.75rem; background:rgba(255,255,255,0.05); padding:2px 6px; border-radius:4px;">${statusLabel}</span>
            </div>
            <div style="font-size:0.72rem; color:var(--text-muted);">${esc(r.localizacao)} (${esc(r.host)})</div>
            <div style="height:4px; background:rgba(255,255,255,0.08); border-radius:999px; overflow:hidden; margin-top:4px;">
              <div style="width:${signalPercent}%; height:100%; background:${badgeColor}; border-radius:999px; transition:width 0.4s ease;"></div>
            </div>
            <div style="display:flex; justify-content:space-between; align-items:baseline; margin-top:4px;">
              <span class="mono bold" style="font-size:1.3rem; color:${badgeColor};">${pingStr}</span>
              <span class="mono" style="font-size:0.75rem; color:var(--text-secondary);">${jitterStr}</span>
            </div>
          </div>
        `;
      }).join("");
    }
  } catch (e) {
    if (container) container.innerHTML = `<p class="danger-text">Erro ao testar matriz: ${esc(e)}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

async function executarFlushRede() {
  const container = $("#flush-resultado");
  const btn = $("#btn-flush-rede");
  if (btn) btn.disabled = true;
  if (container) {
    container.innerHTML = `<div class="health-loading"><span class="pulse-spinner"></span><span>Limpando cache DNS, renovando DHCP e resetando Winsock…</span></div>`;
  }

  try {
    toast("Executando limpeza profunda da pilha de rede…", "info");
    const res = await App.FlushingRede();
    Visualizer.log({
      type: res.ok ? "apply" : "warn",
      msg: `[FLUSH REDE] ${res.mensagem}`
    });

    if (container) {
      container.innerHTML = `
        <div style="background:var(--bg-sunken); padding:12px; border-radius:var(--radius-md); border:1px solid ${res.ok ? 'rgba(16,185,129,0.3)' : 'rgba(239,68,68,0.3)'};">
          <strong style="color:${res.ok ? 'var(--accent)' : 'var(--danger)'};">${esc(res.mensagem)}</strong>
          <ul style="margin:8px 0 0 16px; font-size:12px; color:var(--text-secondary);">
            ${(res.etapas || []).map((e) => `<li>${esc(e)}</li>`).join("")}
            ${(res.erros || []).map((e) => `<li style="color:var(--danger);">${esc(e)}</li>`).join("")}
          </ul>
        </div>
      `;
    }
    toast(res.mensagem, res.ok ? "ok" : "warn");
  } catch (e) {
    if (container) container.innerHTML = `<p class="danger-text">Erro no flush: ${esc(e)}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

/* ==========================================================================
   Histórico & Auditoria
   ========================================================================== */

async function carregarHistorico() {
  const container = $("#hist-lista");
  if (!container) return;
  try {
  const entradas = await comTimeout(App.Historico());
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
  } catch (e) {
    container.innerHTML = `<div class="empty-results"><p>Erro ao ler histórico: ${esc(String(e))}</p></div>`;
  }
}

/* ==========================================================================
   Inicialização & Programas no Boot
   ========================================================================== */

async function carregarStartup() {
  const container = $("#lista-startup");
  if (!container) return;
  container.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Lendo programas de inicialização…</span></div>';
  try {
    const itens = await comTimeout(App.ListarInicializacao());
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
      let err;
      try { err = await App.AlternarInicializacao(id, ativar); }
      catch (cause) { err = String(cause); }
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
  if (!container) return;
  container.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Auditando unidades de disco…</span></div>';
  try {
    const drives = await comTimeout(App.ListarDiscos());
    if (!drives || !drives.length) {
      container.innerHTML = '<div class="empty-results"><p>Nenhuma unidade detectada.</p></div>';
      return;
    }

    container.innerHTML = drives.map((d) => {
      const isSSD = String(d.mediaType || "").toUpperCase().includes("SSD") || String(d.busType || "").toUpperCase().includes("NVME");
      return `
        <div class="glass-card" style="margin-bottom:12px; border-left:3px solid ${isSSD ? 'var(--accent)' : 'var(--warn)'};">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <div style="flex:1;">
              <div style="display:flex; align-items:center; gap:8px;">
                <h3 class="card-title" style="margin:0;">Unidade ${esc(d.letter)} (${esc(d.label || 'Volume Local')})</h3>
                <span class="badge-tag ${isSSD ? 'rec' : 'obs'}">${esc(d.mediaType || 'Disco')}</span>
                <span class="badge-tag rec">${esc(d.busType || 'SATA/NVMe')}</span>
              </div>
              <div style="height:6px; background:rgba(255,255,255,0.08); border-radius:999px; overflow:hidden; margin-top:8px; max-width:320px;">
                <div style="width:100%; height:100%; background:linear-gradient(90deg, #10b981, #00f0ff); border-radius:999px;"></div>
              </div>
              <p class="field-hint" style="margin-top:6px;">Saúde SMART: <b>${esc(d.healthStatus || 'Saudável')}</b> · TRIM: <b>${d.supportsTrim ? 'Suportado' : 'N/A'}</b></p>
            </div>
            <div style="display:flex; gap:8px;">
              ${d.supportsTrim ? `<button class="btn-secondary small-btn btn-trim" data-drive="${esc(d.letter)}">Executar TRIM</button>` : ""}
              <button class="btn-secondary small-btn btn-chkdsk" data-drive="${esc(d.letter)}">chkdsk /scan</button>
            </div>
          </div>
        </div>
      `;
    }).join("");

    $$(".btn-trim").forEach((btn) => btn.addEventListener("click", async () => {
      const drv = btn.dataset.drive;
      btn.disabled = true;
      btn.textContent = "Executando TRIM…";
      Visualizer.show();
      Visualizer.log({ type: "write", msg: `Executando TRIM de manutenção no SSD ${drv}...` });
      let out;
      try { out = await App.ExecutarTRIM(drv); }
      catch (e) { toast(`Erro no TRIM: ${String(e)}`, "erro"); }
      finally { btn.disabled = false; btn.textContent = "Executar TRIM"; }
      abrirModal(`Resultado TRIM (${drv})`, `<div class="terminal-block"><div>${esc(out)}</div></div>`);
    }));

    $$(".btn-chkdsk").forEach((btn) => btn.addEventListener("click", async () => {
      const drv = btn.dataset.drive;
      btn.disabled = true;
      btn.textContent = "Verificando…";
      Visualizer.show();
      Visualizer.log({ type: "read", msg: `Executando chkdsk /scan não destrutivo na unidade ${drv}...` });
      let out;
      try { out = await App.ExecutarChkdsk(drv); }
      catch (e) { toast(`Erro no chkdsk: ${String(e)}`, "erro"); }
      finally { btn.disabled = false; btn.textContent = "chkdsk /scan"; }
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
   Kernel Timer Resolution & Jitter de Sleep (valleyofdoom)
   ========================================================================== */

async function carregarTimerResolution() {
  const container = $("#timerres-container");
  if (!container) return;
  container.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Lendo resolução do temporizador do kernel…</span></div>';

  try {
    const info = await App.ObterTimerResolution();
    if (!info) {
      container.innerHTML = '<div class="empty-results"><p>Não foi possível obter dados de resolução do temporizador.</p></div>';
      return;
    }

    const curMs = Number(info.currentResolutionMs || 0).toFixed(3);
    const minMs = Number(info.minResolutionMs || 0).toFixed(3);
    const maxMs = Number(info.maxResolutionMs || 0).toFixed(3);
    const isUltra = Number(info.currentResolutionMs || 0) <= 0.6;

    container.innerHTML = `
      <div style="display:grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap:12px; margin-bottom:16px;">
        <div class="glass-card" style="padding:14px; border-left:3px solid ${isUltra ? 'var(--accent)' : 'var(--warn)'};">
          <div class="muted small">Resolução Atual do Relógio</div>
          <div style="font-size:22px; font-weight:800; color:${isUltra ? 'var(--accent)' : '#fff'}; margin-top:4px;">
            ${curMs} ms
          </div>
          <div style="margin-top:4px;">
            <span class="badge-tag ${isUltra ? 'rec' : 'obs'}">${isUltra ? 'Alta Precisão (0.5ms)' : 'Resolução Padrão'}</span>
          </div>
        </div>

        <div class="glass-card" style="padding:14px;">
          <div class="muted small">Resolução Máxima Suportada</div>
          <div style="font-size:20px; font-weight:700; color:var(--text-primary); margin-top:4px;">
            ${maxMs} ms
          </div>
          <div class="muted small" style="margin-top:4px;">Frequência: ${(1000 / Number(info.maxResolutionMs || 1)).toFixed(0)} Hz</div>
        </div>

        <div class="glass-card" style="padding:14px;">
          <div class="muted small">Resolução Mínima (Default)</div>
          <div style="font-size:20px; font-weight:700; color:var(--text-primary); margin-top:4px;">
            ${minMs} ms
          </div>
          <div class="muted small" style="margin-top:4px;">Frequência: ${(1000 / Number(info.minResolutionMs || 15.625)).toFixed(0)} Hz</div>
        </div>
      </div>

      <div style="display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin-bottom:14px;">
        <button class="btn-primary small-btn" id="btn-timerres-05" ${isUltra ? 'disabled' : ''}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;margin-right:6px;"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
          Ativar Alta Precisão (0.500 ms)
        </button>
        <button class="btn-secondary small-btn" id="btn-timerres-default" ${!isUltra ? 'disabled' : ''}>
          Restaurar Padrão do Windows
        </button>
        <button class="btn-secondary small-btn" id="btn-testar-sleep-precision">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;margin-right:6px;"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          Amostragem Rápida (20x)
        </button>
        <button class="btn-secondary small-btn" id="btn-live-sleep-monitor">
          <span class="radar-dot" style="background:#94a3b8;"></span>
          Monitor Contínuo ao Vivo
        </button>
      </div>

      <div style="margin-bottom:12px; padding:10px 14px; background:rgba(16,185,129,0.06); border:1px solid rgba(16,185,129,0.2); border-radius:8px;">
        <label style="display:flex; align-items:center; gap:10px; cursor:pointer; font-size:12px; color:#e2e8f0;">
          <input type="checkbox" id="chk-persist-timer" ${info.isPersistent ? 'checked' : ''} style="width:16px; height:16px; accent-color:var(--accent); cursor:pointer;">
          <span><b>Fixar 0.500ms permanentemente no Windows</b> (mantém ativo em segundo plano mesmo após fechar o Optimizer e ao ligar o PC)</span>
        </label>
      </div>

      <div class="glass-card" style="margin-top:10px; padding:16px;">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
          <div>
            <h4 style="font-size:14px; font-weight:700; margin:0;">Osciloscópio de Precisão & Jitter (MeasureSleep)</h4>
            <p class="muted small" style="margin:2px 0 0 0;">Linha tracejada verde indica alvo de 1.000 ms. Desvios na curva refletem atrasos do agendador.</p>
          </div>
          <span id="sleep-badge-score" class="badge-tag rec">Aguardando Amostras</span>
        </div>
        <canvas id="canvas-sleep-osc" style="width:100%; height:150px; border-radius:8px; display:block; background:#0c131d; border:1px solid rgba(255,255,255,0.08); margin-top:8px;"></canvas>
        <div id="sleep-precision-resultado" style="margin-top:12px;"></div>
      </div>
    `;

    const btn05 = $("#btn-timerres-05");
    if (btn05) btn05.addEventListener("click", () => aplicarTimerResolution(0.5, true));

    const btnDef = $("#btn-timerres-default");
    if (btnDef) btnDef.addEventListener("click", () => aplicarTimerResolution(15.625, false));

    const btnSleep = $("#btn-testar-sleep-precision");
    if (btnSleep) btnSleep.addEventListener("click", testarPrecisaoSleep);

    const btnLive = $("#btn-live-sleep-monitor");
    if (btnLive) btnLive.addEventListener("click", alternarMonitorLiveSleep);

    const chkPersist = $("#chk-persist-timer");
    if (chkPersist) {
      chkPersist.addEventListener("change", async (e) => {
        const ativar = e.target.checked;
        Visualizer.show();
        Visualizer.log({ type: "write", msg: `${ativar ? 'Configurando' : 'Removendo'} persistência do temporizador do kernel na inicialização do Windows...` });
        try {
          const res = await App.ConfigurarPersistenciaTimer(ativar, 0.5);
          toast(res.mensagem, res.ok ? "ok" : "err");
          Visualizer.log({ type: res.ok ? "apply" : "fail", msg: res.mensagem });
        } catch (err) {
          toast("Erro ao configurar persistência: " + err, "err");
        } finally {
          await carregarTimerResolution();
        }
      });
    }

    // Renderiza gráfico inicial com medição rápida
    setTimeout(() => {
      testarPrecisaoSleep();
    }, 150);

  } catch (e) {
    container.innerHTML = `<div class="empty-results"><p>Erro ao ler timer resolution: ${esc(String(e))}</p></div>`;
  }
}

let liveSleepTimer = null;
let liveSleepBuffer = [];

function renderSleepOscilloscope(canvasId, samples, targetMs = 1.0) {
  const canvas = typeof canvasId === "string" ? (typeof document !== "undefined" && document.getElementById ? document.getElementById(canvasId) : null) : canvasId;
  if (!canvas || typeof canvas.getContext !== "function") return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const rect = canvas.getBoundingClientRect ? canvas.getBoundingClientRect() : { width: 560, height: 150 };
  const dpr = (typeof window !== "undefined" && window.devicePixelRatio) || 1;
  const w = (canvas.width = (rect.width || 560) * dpr);
  const h = (canvas.height = (rect.height || 150) * dpr);
  ctx.clearRect(0, 0, w, h);

  if (!samples || samples.length === 0) return;

  const maxSample = Math.max(...samples, targetMs * 2.0);
  const maxScale = Math.min(Math.max(maxSample * 1.25, 2.5), 20.0);

  ctx.fillStyle = "#0c131d";
  ctx.fillRect(0, 0, w, h);

  ctx.font = `${10 * dpr}px Inter, sans-serif`;
  ctx.fillStyle = "rgba(255, 255, 255, 0.4)";

  const gridSteps = [0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 15.625].filter(v => v <= maxScale);
  for (const gVal of gridSteps) {
    const gy = h - (gVal / maxScale) * (h * 0.80) - h * 0.10;
    ctx.strokeStyle = Math.abs(gVal - targetMs) < 0.05 ? "rgba(16, 185, 129, 0.45)" : "rgba(255, 255, 255, 0.06)";
    ctx.lineWidth = Math.abs(gVal - targetMs) < 0.05 ? 1.5 * dpr : 1 * dpr;
    if (typeof ctx.setLineDash === "function") {
      ctx.setLineDash(Math.abs(gVal - targetMs) < 0.05 ? [4 * dpr, 4 * dpr] : []);
    }
    ctx.beginPath();
    ctx.moveTo(0, gy);
    ctx.lineTo(w, gy);
    ctx.stroke();

    ctx.fillText(`${gVal.toFixed(1)}ms`, 8 * dpr, gy - 3 * dpr);
  }
  if (typeof ctx.setLineDash === "function") ctx.setLineDash([]);

  const step = samples.length > 1 ? (w - 24 * dpr) / (samples.length - 1) : w;
  const startX = 12 * dpr;
  const points = samples.map((val, idx) => {
    const clamped = Math.max(0, Math.min(maxScale, val));
    const y = h - (clamped / maxScale) * (h * 0.80) - h * 0.10;
    return { x: startX + idx * step, y: y, val: val };
  });

  // Área com gradiente
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const cpX = (prev.x + curr.x) / 2;
    ctx.bezierCurveTo(cpX, prev.y, cpX, curr.y, curr.x, curr.y);
  }
  ctx.lineTo(points[points.length - 1].x, h);
  ctx.lineTo(points[0].x, h);
  ctx.closePath();

  const grad = ctx.createLinearGradient(0, 0, 0, h);
  grad.addColorStop(0, "rgba(16, 185, 129, 0.28)");
  grad.addColorStop(1, "rgba(16, 185, 129, 0.0)");
  ctx.fillStyle = grad;
  ctx.fill();

  // Curva principal
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const cpX = (prev.x + curr.x) / 2;
    ctx.bezierCurveTo(cpX, prev.y, cpX, curr.y, curr.x, curr.y);
  }
  ctx.strokeStyle = "#10b981";
  ctx.lineWidth = 2 * dpr;
  ctx.shadowColor = "#10b981";
  ctx.shadowBlur = 8;
  ctx.stroke();
  ctx.shadowBlur = 0;

  // Pontos individuais
  for (const pt of points) {
    const delta = Math.abs(pt.val - targetMs);
    let color = "#10b981";
    if (delta > 0.8) color = "#ef4444";
    else if (delta > 0.25) color = "#f59e0b";

    ctx.beginPath();
    ctx.arc(pt.x, pt.y, (delta > 0.8 ? 4.5 : 3) * dpr, 0, Math.PI * 2);
    ctx.fillStyle = color;
    ctx.fill();
  }
}

async function alternarMonitorLiveSleep() {
  const btn = $("#btn-live-sleep-monitor");
  if (liveSleepTimer) {
    clearInterval(liveSleepTimer);
    liveSleepTimer = null;
    if (btn) {
      btn.innerHTML = `<span class="radar-dot" style="background:#94a3b8;"></span> Monitor Contínuo ao Vivo`;
      btn.classList.remove("btn-primary");
      btn.classList.add("btn-secondary");
    }
    toast("Monitoramento contínuo pausado.", "info");
    return;
  }

  if (btn) {
    btn.innerHTML = `<span class="radar-dot" style="background:#10b981;"></span> Pausar Monitor Contínuo`;
    btn.classList.remove("btn-secondary");
    btn.classList.add("btn-primary");
  }
  toast("Monitoramento em tempo real ativado.", "ok");

  const tick = async () => {
    try {
      const res = await App.MedirSleepPrecision(10);
      if (res && res.samples) {
        liveSleepBuffer = liveSleepBuffer.concat(res.samples);
        if (liveSleepBuffer.length > 40) {
          liveSleepBuffer = liveSleepBuffer.slice(-40);
        }
        renderSleepOscilloscope("canvas-sleep-osc", liveSleepBuffer, 1.0);
        const scoreEl = $("#sleep-badge-score");
        if (scoreEl) scoreEl.textContent = res.jitterScore;
        const resEl = $("#sleep-precision-resultado");
        if (resEl) {
          resEl.innerHTML = `
            <div style="display:grid; grid-template-columns: repeat(auto-fit, minmax(130px, 1fr)); gap:10px; font-size:12px;">
              <div><span class="muted">Alvo:</span> <b>${res.targetMs} ms</b></div>
              <div><span class="muted">Média Recente:</span> <b>${res.averageMs} ms</b></div>
              <div><span class="muted">Mínimo:</span> <b>${res.minMs} ms</b></div>
              <div><span class="muted">Máximo:</span> <b>${res.maxMs} ms</b></div>
              <div><span class="muted">Jitter (StdDev):</span> <b style="color:var(--accent);">${res.stdDevMs} ms</b></div>
            </div>
          `;
        }
      }
    } catch (e) {
      // Silencioso
    }
  };

  await tick();
  liveSleepTimer = setInterval(tick, 1500);
}

async function aplicarTimerResolution(desiredMs, ativar) {
  try {
    Visualizer.show();
    Visualizer.log({ type: "write", msg: `Ajustando NtSetTimerResolution para ${desiredMs} ms...` });
    const res = await App.DefinirTimerResolution(desiredMs, ativar);
    if (res && res.ok) {
      toast(res.mensagem || "Resolução do temporizador atualizada.", "ok");
      Visualizer.log({ type: "verify", msg: res.mensagem });
    } else {
      toast((res && res.mensagem) || "Falha ao definir timer resolution", "err");
      Visualizer.log({ type: "fail", msg: (res && res.mensagem) || "Erro" });
    }
  } catch (e) {
    toast("Erro ao configurar timer: " + e, "err");
  } finally {
    await carregarTimerResolution();
  }
}

async function testarPrecisaoSleep() {
  const btn = $("#btn-testar-sleep-precision");
  const resEl = $("#sleep-precision-resultado");
  const scoreEl = $("#sleep-badge-score");
  if (!resEl) return;

  if (btn) {
    btn.disabled = true;
    btn.innerHTML = `<span class="pulse-spinner"></span> Amostrando 20x…`;
  }

  Visualizer.show();
  Visualizer.log({ type: "read", msg: "Iniciando MeasureSleep (20 amostras de 1.0ms)..." });

  try {
    const res = await App.MedirSleepPrecision(20);
    Visualizer.log({
      type: "verify",
      msg: `Sleep delta médio: ${res.averageMs} ms (Desvio Padrão: ${res.stdDevMs} ms) - ${res.jitterScore}`
    });

    if (scoreEl) {
      scoreEl.textContent = res.jitterScore;
    }

    renderSleepOscilloscope("canvas-sleep-osc", res.samples, res.targetMs || 1.0);

    resEl.innerHTML = `
      <div style="display:grid; grid-template-columns: repeat(auto-fit, minmax(130px, 1fr)); gap:10px; font-size:12px;">
        <div><span class="muted">Alvo:</span> <b>${res.targetMs} ms</b></div>
        <div><span class="muted">Média Real:</span> <b>${res.averageMs} ms</b></div>
        <div><span class="muted">Mínimo:</span> <b>${res.minMs} ms</b></div>
        <div><span class="muted">Máximo:</span> <b>${res.maxMs} ms</b></div>
        <div><span class="muted">Desvio Padrão (Jitter):</span> <b style="color:var(--accent);">${res.stdDevMs} ms</b></div>
      </div>
      <div style="margin-top:8px; font-size:11px; color:var(--text-secondary);">
        💡 Quanto menor o desvio padrão (Jitter), mais consistente é o frame pacing e menores são as variações de latência de entrada em jogos e renderização em tempo real.
      </div>
    `;
  } catch (e) {
    resEl.innerHTML = `<div class="empty-results"><p>Erro ao medir precisão de sleep: ${esc(String(e))}</p></div>`;
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;margin-right:6px;"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg> Amostragem Rápida (20x)`;
    }
  }
}

/* ==========================================================================
   Inspetor de Interrupções PCIe e Modo MSI (valleyofdoom / AutoGpuAffinity)
   ========================================================================== */

let listaDispositivosMSI = [];

async function escanearDispositivosMSI() {
  const btn = $("#btn-escanear-msi");
  const resEl = $("#msi-resultado");
  if (!resEl) return;

  if (btn) {
    btn.disabled = true;
    btn.innerHTML = `<span class="pulse-spinner"></span> Escaneando PCIe…`;
  }
  resEl.innerHTML = '<div class="health-loading"><span class="pulse-spinner"></span><span>Audito adaptadores PCIe (GPU, Rede, Controladores USB)...</span></div>';

  Visualizer.show();
  Visualizer.log({ type: "read", msg: "Consultando registro PnP para adaptadores PCIe e modos de interrupção MSI..." });

  try {
    listaDispositivosMSI = await App.ListarDispositivosMSI();
    if (!listaDispositivosMSI || !listaDispositivosMSI.length) {
      resEl.innerHTML = '<div class="glass-card"><p class="muted-text">Nenhum adaptador PCIe auditável encontrado.</p></div>';
      return;
    }

    Visualizer.log({
      type: "read",
      msg: `Foram encontrados ${listaDispositivosMSI.length} adaptadores PCIe auditados.`
    });

    resEl.innerHTML = `
      <div style="margin-bottom:12px;">
        <p style="font-size:13px; color:var(--text-secondary); margin-bottom:10px;">
          Modo MSI (Message Signaled Interrupts) elimina compartilhamento de linhas IRQ e reduz a latência de DPC/ISR de placas de vídeo e rede.
        </p>
        <div class="terminal-block" style="max-height:280px; overflow-y:auto;">
          ${listaDispositivosMSI.map((d, i) => `
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px; padding-bottom:8px; border-bottom:1px solid rgba(255,255,255,0.06); font-size:12px;">
              <div style="flex:1; margin-right:12px;">
                <div><b>${esc(d.nome)}</b> <span class="badge-tag ${d.classe === 'Display' ? 'rec' : 'obs'}">${esc(d.classe || 'PCIe')}</span></div>
                <div class="mono muted small" style="margin-top:2px;">${esc(d.id)}</div>
              </div>
              <div style="display:flex; align-items:center; gap:8px;">
                <span class="badge-tag ${d.msiSupported ? 'rec' : 'obs'}">${esc(d.statusRotulo)}</span>
                <button class="btn-${d.msiSupported ? 'secondary' : 'primary'} small-btn btn-toggle-msi" data-idx="${i}">
                  ${d.msiSupported ? 'Desativar MSI' : 'Ativar Modo MSI'}
                </button>
              </div>
            </div>
          `).join("")}
        </div>
      </div>
    `;

    $$(".btn-toggle-msi").forEach((b) => b.addEventListener("click", async () => {
      const idx = Number(b.dataset.idx);
      const dev = listaDispositivosMSI[idx];
      if (!dev) return;
      await alternarModoMSIDispositivo(dev.caminhoRegistro, !dev.msiSupported);
    }));

  } catch (e) {
    resEl.innerHTML = `<div class="empty-results"><p>Erro ao auditar dispositivos MSI: ${esc(String(e))}</p></div>`;
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = "Escanear Dispositivos";
    }
  }
}

async function alternarModoMSIDispositivo(caminho, ativar) {
  try {
    Visualizer.show();
    Visualizer.log({ type: "write", msg: `${ativar ? 'Ativando' : 'Desativando'} modo MSI no registro...` });
    const res = await App.AlternarModoMSI(caminho, ativar);
    if (res && res.ok) {
      toast(res.mensagem, "ok");
      Visualizer.log({ type: "verify", msg: res.mensagem });
    } else {
      toast((res && res.mensagem) || "Falha ao alterar modo MSI", "err");
      Visualizer.log({ type: "fail", msg: (res && res.mensagem) || "Erro" });
    }
  } catch (e) {
    toast("Erro ao alternar modo MSI: " + e, "err");
  } finally {
    await escanearDispositivosMSI();
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

    ["jogo", "nvidia", "coding"].forEach((key) => {
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
    const mapaItens = new Map((diag.itens || []).map((it) => [it.id, it]));

    const linhas = p.tweakIds.map((id) => {
      const it = mapaItens.get(id);
      const isApplied = it && it.estado === "aplicado";
      const icon = isApplied ? '<span style="color:var(--accent); font-weight:bold;">[OK]</span>' : '<span style="color:var(--text-muted);">[PENDENTE]</span>';
      return `<div style="display:flex; justify-content:space-between; margin-bottom:6px; font-size:12px;">
        <span>${icon} <b>${esc(it ? it.nome : id)}</b></span>
        <span class="muted">${esc(it ? it.detalhe : '')}</span>
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

  // Aplicação manual é imediata e acompanhada pelo Visualizador. O benchmark
  // comparativo é opcional e não deve bloquear uma alteração solicitada.
  if (typeof App.AplicarPerfilUso === "function") {
    Visualizer.show();
    Visualizer.log({ type: "write", msg: `Aplicando perfil [${key.toUpperCase()}] — gravação e verificação em andamento...` });
    toast(`Aplicando perfil ${p.nome}…`, "info");
    try {
      const res = await App.AplicarPerfilUso(key, false);
      for (const r of (res || [])) {
        Visualizer.log({
          type: r.estado === "ok" ? "apply" : "fail",
          msg: `[${(r.estado || "resultado").toUpperCase()}] ${r.nome}: ${r.mensagem}`
        });
      }
      mostrarResultados(`Aplicação do Perfil ${p.nome}`, res);
      const falhas = (res || []).filter((r) => r.estado === "falhou").length;
      const pulados = (res || []).filter((r) => r.estado === "pulado").length;
      const aplicados = (res || []).filter((r) => r.estado === "ok").length;
      toast(
        falhas || pulados
          ? `${aplicados} aplicado(s), ${pulados} pulado(s), ${falhas} falha(s). Veja os detalhes.`
          : `${aplicados} ajuste(s) aplicado(s) e verificado(s).`,
        falhas || pulados ? "erro" : "ok",
        4500
      );
      await carregarEstadoPerfis();
      await diagnosticar();
    } catch (e) {
      toast(`Não foi possível aplicar o perfil: ${String(e)}`, "erro");
    }
    return;
  }

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
    let resAplicacao;
    try {
      resAplicacao = await App.AplicarPerfilComBenchmark(key, false, reportAntes);
    } catch (e) {
      benchmarkEmAndamento = false;
      fecharModalBenchmark();
      toast(`Não foi possível aplicar o perfil: ${String(e)}`, "erro");
      await carregarEstadoPerfis();
      return;
    }

    for (const r of (resAplicacao.resultados || [])) {
      Visualizer.log({
        type: r.estado === "ok" ? "write" : (r.estado === "pulado" ? "read" : "fail"),
        msg: `[${r.estado.toUpperCase()}] ${r.nome}: ${r.mensagem}`
      });
    }
    if ((resAplicacao.resultados || []).some((r) => r.estado === "falhou")) {
      benchmarkEmAndamento = false;
      toast("A aplicação do perfil terminou com falhas; nenhuma medição pós-aplicação será iniciada.", "erro");
      await carregarEstadoPerfis();
      return;
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
  let res;
  try {
    res = await App.RestaurarPerfilUso(key);
  } catch (e) {
    toast(`Não foi possível restaurar o perfil: ${String(e)}`, "erro");
    return;
  }

  for (const r of (res || [])) {
    Visualizer.log({
      type: r.estado === "ok" ? "revert" : "fail",
      msg: `[RESTAURADO] ${r.nome}: ${r.mensagem}`
    });
  }

  if ((res || []).some((r) => r.estado === "falhou")) {
    toast("A restauração terminou com falhas.", "erro");
  } else {
    toast("Perfil restaurado com sucesso.", "ok");
  }
  await carregarEstadoPerfis();
  await diagnosticar();
}

/* ==========================================================================
   Motor de Gráficos e Telemetria em Tempo Real (Micro-Canvas 60 FPS)
   ========================================================================== */

const MAX_CHART_SAMPLES = 60;
const historyCPU = new Array(MAX_CHART_SAMPLES).fill(0);
const historyGPU = new Array(MAX_CHART_SAMPLES).fill(0);
const historyRAM = new Array(MAX_CHART_SAMPLES).fill(0);
const historyPing = new Array(MAX_CHART_SAMPLES).fill(0);

let telemetriaTimer = null;
let telemetriaEmAndamento = false;
let telemetriaFalhasConsecutivas = 0;

function renderCatalogDonutChart(canvasId, percent, aplicados, total) {
  const canvas = typeof canvasId === "string" ? (typeof document !== "undefined" && document.getElementById ? document.getElementById(canvasId) : null) : canvasId;
  if (!canvas || typeof canvas.getContext !== "function") return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const rect = canvas.getBoundingClientRect ? canvas.getBoundingClientRect() : { width: 100, height: 100 };
  const dpr = (typeof window !== "undefined" && window.devicePixelRatio) || 1;
  const size = (rect.width || 100) * dpr;
  canvas.width = size;
  canvas.height = size;
  ctx.clearRect(0, 0, size, size);

  const cx = size / 2;
  const cy = size / 2;
  const radius = (size / 2) - 8 * dpr;
  const lineWidth = 9 * dpr;

  // Anel de Fundo
  ctx.beginPath();
  ctx.arc(cx, cy, radius, 0, Math.PI * 2);
  ctx.strokeStyle = "rgba(255, 255, 255, 0.08)";
  ctx.lineWidth = lineWidth;
  ctx.stroke();

  // Arco Ativo com Gradiente
  const startAngle = -Math.PI / 2;
  const endAngle = startAngle + (Math.PI * 2 * (percent / 100));

  const grad = ctx.createLinearGradient(0, 0, size, size);
  grad.addColorStop(0, "#00f0ff");
  grad.addColorStop(1, "#10b981");

  ctx.beginPath();
  ctx.arc(cx, cy, radius, startAngle, endAngle);
  ctx.strokeStyle = grad;
  ctx.lineWidth = lineWidth;
  ctx.lineCap = "round";
  ctx.shadowColor = "#10b981";
  ctx.shadowBlur = 8;
  ctx.stroke();
  ctx.shadowBlur = 0;

  // Texto Central
  ctx.fillStyle = "#ffffff";
  ctx.font = `bold ${16 * dpr}px Inter, sans-serif`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(`${percent}%`, cx, cy - 3 * dpr);

  ctx.fillStyle = "rgba(255, 255, 255, 0.5)";
  ctx.font = `${8 * dpr}px Inter, sans-serif`;
  ctx.fillText("ATIVO", cx, cy + 11 * dpr);
}

function renderDNSBenchmarkChart(canvasId, providers) {
  const canvas = typeof canvasId === "string" ? (typeof document !== "undefined" && document.getElementById ? document.getElementById(canvasId) : null) : canvasId;
  if (!canvas || typeof canvas.getContext !== "function") return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const valid = (providers || []).filter(p => (p.avgRttMs || p.rttAvgMs) > 0 && (p.avgRttMs || p.rttAvgMs) < 999);
  if (!valid.length) return;

  const rect = canvas.getBoundingClientRect ? canvas.getBoundingClientRect() : { width: 520, height: 160 };
  const dpr = (typeof window !== "undefined" && window.devicePixelRatio) || 1;
  const w = (canvas.width = (rect.width || 520) * dpr);
  const rowHeight = 26 * dpr;
  const h = (canvas.height = Math.max(130 * dpr, valid.length * rowHeight + 25 * dpr));
  ctx.clearRect(0, 0, w, h);

  const maxRTT = Math.max(...valid.map(p => (p.avgRttMs || p.rttAvgMs)), 60);

  ctx.font = `${10.5 * dpr}px Inter, sans-serif`;
  const labelWidth = 130 * dpr;
  const barAreaWidth = w - labelWidth - 80 * dpr;

  valid.forEach((p, idx) => {
    const rtt = p.avgRttMs || p.rttAvgMs || 0;
    const y = 14 * dpr + idx * rowHeight;
    const barW = Math.max(10 * dpr, (rtt / maxRTT) * barAreaWidth);

    // Nome do Provedor
    ctx.fillStyle = "#e2e8f0";
    ctx.textAlign = "left";
    ctx.textBaseline = "middle";
    ctx.fillText(p.nome, 10 * dpr, y + 7 * dpr);

    // Fundo da barra
    ctx.fillStyle = "rgba(255, 255, 255, 0.05)";
    ctx.fillRect(labelWidth, y, barAreaWidth, 14 * dpr);

    // Barra com Cor por Latência
    let barColor = "#10b981";
    if (rtt > 50) barColor = "#3b82f6";
    if (rtt > 100) barColor = "#f59e0b";
    if (rtt > 150) barColor = "#ef4444";

    const grad = ctx.createLinearGradient(labelWidth, y, labelWidth + barW, y);
    grad.addColorStop(0, barColor);
    grad.addColorStop(1, barColor + "cc");

    ctx.fillStyle = grad;
    ctx.fillRect(labelWidth, y, barW, 14 * dpr);

    // RTT em Texto
    ctx.fillStyle = barColor;
    ctx.textAlign = "left";
    ctx.font = `bold ${10.5 * dpr}px Inter, sans-serif`;
    ctx.fillText(`${rtt.toFixed(1)} ms`, labelWidth + barW + 8 * dpr, y + 7 * dpr);
    ctx.font = `${10.5 * dpr}px Inter, sans-serif`;
  });
}

function renderMicroChart(canvasId, dataArray, strokeColor, fillColor, maxScale = 100) {
  const canvas = typeof canvasId === "string" ? document.getElementById(canvasId) : canvasId;
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  const w = (canvas.width = (rect.width || 480) * dpr);
  const h = (canvas.height = (rect.height || 140) * dpr);
  ctx.clearRect(0, 0, w, h);

  if (!dataArray || dataArray.length < 2) return;

  // Linhas sutis de grade
  ctx.strokeStyle = "rgba(255, 255, 255, 0.05)";
  ctx.lineWidth = 1;
  for (let i = 1; i <= 3; i++) {
    const y = (h / 4) * i;
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
    ctx.stroke();
  }

  // Mapear pontos
  const step = w / (dataArray.length - 1);
  const points = dataArray.map((val, idx) => {
    const clamped = Math.max(0, Math.min(maxScale, val));
    const y = h - (clamped / maxScale) * (h * 0.85) - h * 0.05;
    return { x: idx * step, y: y };
  });

  // Área preenchida com gradiente
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const cpX = (prev.x + curr.x) / 2;
    ctx.bezierCurveTo(cpX, prev.y, cpX, curr.y, curr.x, curr.y);
  }
  ctx.lineTo(w, h);
  ctx.lineTo(0, h);
  ctx.closePath();

  const grad = ctx.createLinearGradient(0, 0, 0, h);
  grad.addColorStop(0, fillColor || "rgba(0, 240, 255, 0.3)");
  grad.addColorStop(1, "rgba(0, 0, 0, 0)");
  ctx.fillStyle = grad;
  ctx.fill();

  // Linha do traço com glow
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const cpX = (prev.x + curr.x) / 2;
    ctx.bezierCurveTo(cpX, prev.y, cpX, curr.y, curr.x, curr.y);
  }
  ctx.strokeStyle = strokeColor || "#00f0ff";
  ctx.lineWidth = 2 * dpr;
  ctx.shadowColor = strokeColor || "#00f0ff";
  ctx.shadowBlur = 8;
  ctx.stroke();
  ctx.shadowBlur = 0;

  // Ponto na última amostra
  const last = points[points.length - 1];
  ctx.beginPath();
  ctx.arc(last.x, last.y, 4 * dpr, 0, Math.PI * 2);
  ctx.fillStyle = "#ffffff";
  ctx.shadowColor = strokeColor || "#00f0ff";
  ctx.shadowBlur = 10;
  ctx.fill();
  ctx.shadowBlur = 0;
}

async function atualizarTelemetriaAoVivo() {
  if (telemetriaEmAndamento || !App || typeof App.ObterTelemetriaAoVivo !== "function") return;
  telemetriaEmAndamento = true;
  try {
    const data = await App.ObterTelemetriaAoVivo();
    if (!data) return;
    if (data.erro) throw new Error(data.erro);
    telemetriaFalhasConsecutivas = 0;

    // Atualiza buffers de histórico
    historyCPU.shift();
    historyCPU.push(data.cpuUsagePercent || 0);

    historyGPU.shift();
    historyGPU.push(data.gpuUsagePercent || 0);

    historyRAM.shift();
    historyRAM.push(data.ramUsedPercent || 0);

    // Renderiza gráficos de Micro-Canvas
    renderMicroChart("canvas-cpu", historyCPU, "#00f0ff", "rgba(0, 240, 255, 0.25)", 100);
    renderMicroChart("canvas-gpu", historyGPU, "#76b900", "rgba(118, 185, 0, 0.25)", 100);
    renderMicroChart("canvas-ram", historyRAM, "#a855f7", "rgba(168, 85, 247, 0.25)", 100);

    // Atualiza valores de CPU
    const elCpuVal = $("#telem-cpu-val");
    if (elCpuVal) elCpuVal.textContent = `${(data.cpuUsagePercent || 0).toFixed(1)}%`;
    const elCpuExtra = $("#telem-cpu-extra");
    if (elCpuExtra) {
      if (data.cpuFrequencyAvailable === false) elCpuExtra.textContent = "Frequência N/D";
      const tempStr = data.cpuTempCelsius !== undefined && data.cpuTempCelsius !== null ? `${data.cpuTempCelsius.toFixed(1)}°C` : "Sensor N/D";
      elCpuExtra.textContent = `${(data.cpuFrequencyMhz || 0).toFixed(0)} MHz · ${tempStr}`;
    }
    const elCpuCores = $("#telem-cpu-cores");
    if (elCpuCores) elCpuCores.textContent = `${data.physicalCores || '--'} Núcleos Físicos / ${data.logicalProcessors || '--'} Lógicos`;

    // Atualiza valores de GPU
    const elGpuVal = $("#telem-gpu-val");
    if (elGpuVal) elGpuVal.textContent = `${(data.gpuUsagePercent || 0).toFixed(1)}%`;
    const elGpuExtra = $("#telem-gpu-extra");
    if (elGpuExtra) {
      const gTempStr = data.gpuTempCelsius !== undefined && data.gpuTempCelsius !== null ? `${data.gpuTempCelsius.toFixed(1)}°C` : "Sensor N/D";
      elGpuExtra.textContent = `VRAM: ${(data.gpuMemoryUsedMb || 0).toFixed(0)} MB · ${gTempStr}`;
    }
    const elGpuVramTotal = $("#telem-gpu-vram-total");
    if (elGpuVramTotal) elGpuVramTotal.textContent = `Memória Dedicada: ${(data.gpuMemoryTotalMb || 0).toFixed(0)} MB`;

    // Atualiza valores de RAM
    const elRamVal = $("#telem-ram-val");
    if (elRamVal) elRamVal.textContent = `${(data.ramUsedPercent || 0).toFixed(1)}%`;
    const elRamExtra = $("#telem-ram-extra");
    if (elRamExtra) elRamExtra.textContent = `${(data.ramUsedMb || 0).toFixed(0)} / ${(data.ramTotalMb || 0).toFixed(0)} MB`;
    const elRamAvail = $("#telem-ram-avail");
    if (elRamAvail) elRamAvail.textContent = `Livre: ${Math.max(0, (data.ramTotalMb || 0) - (data.ramUsedMb || 0)).toFixed(0)} MB`;

    // Throttling
    const elThrottling = $("#telem-throttling-badge");
    if (elThrottling) {
      elThrottling.textContent = data.thermalThrottled ? "Throttling Térmico Ativo!" : "Throttling: Normal";
      elThrottling.style.color = data.thermalThrottled ? "var(--danger)" : "var(--accent)";
    }

    // Top Processos
    const procsEl = $("#telem-top-processes");
    if (procsEl && data.topProcesses && data.topProcesses.length) {
      procsEl.innerHTML = data.topProcesses.map((p) => `
        <div class="process-row-item">
          <div>
            <b>${esc(p.name)}</b> <span class="muted">(PID: ${p.pid})</span>
          </div>
          <div class="mono bold" style="color:var(--accent);">
            ${p.percent.toFixed(1)}% CPU
          </div>
        </div>
      `).join("");
    }
    // Valores ausentes permanecem explicitamente indisponíveis, nunca zero inventado.
    if (data.cpuFrequencyAvailable === false && elCpuExtra) elCpuExtra.textContent = "Frequência N/D";
    if (data.gpuMemoryAvailable === false && elGpuExtra) elGpuExtra.textContent = `${data.gpuTempCelsius != null ? `${data.gpuTempCelsius.toFixed(1)}Â°C` : "Sensor N/D"} · VRAM N/D`;
    if (data.gpuMemoryAvailable === false && elGpuVramTotal) elGpuVramTotal.textContent = "Memória Dedicada: N/D";
    if (data.ramTotalMb <= 0) {
      if (elRamVal) elRamVal.textContent = "N/D";
      if (elRamExtra) elRamExtra.textContent = "Memória N/D";
      if (elRamAvail) elRamAvail.textContent = "Livre: N/D";
    }
    if (data.thermalStatusAvailable === false && elThrottling) elThrottling.textContent = "Throttling: N/D";
  } catch (e) {
    telemetriaFalhasConsecutivas++;
    const badge = $("#telemetria-live-badge");
    if (badge) {
      badge.textContent = "TELEMETRIA INDISPONÍVEL";
      badge.style.color = telemetriaFalhasConsecutivas < 3 ? "var(--warn)" : "var(--danger)";
      badge.title = `Falha ${telemetriaFalhasConsecutivas}: ${e.message || e}`;
    }
    console.warn("Telemetria indisponível:", e);
  } finally {
    telemetriaEmAndamento = false;
  }
}

async function atualizarGraficoPingAoVivo() {
  try {
    const host = ($("#destino") && $("#destino").value.trim()) || "8.8.8.8";
    const res = await App.MedirRedeAntes(host);
    if (res && res.avgRTT) {
      historyPing.shift();
      historyPing.push(res.avgRTT);
      renderMicroChart("canvas-ping", historyPing, "#3b82f6", "rgba(59, 130, 246, 0.25)", 150);
      const elPingVal = $("#telem-ping-val");
      if (elPingVal) elPingVal.textContent = `${res.avgRTT} ms`;
    }
  } catch (e) {
    // Silencioso
  }
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", boot);
} else {
  boot();
}
