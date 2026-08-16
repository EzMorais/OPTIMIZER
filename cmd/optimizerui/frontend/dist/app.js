// Interface do Optimizer. Nenhuma decisão mora aqui: tudo que é diagnóstico,
// risco ou alteração vem do Go compilado. Este arquivo só pede e mostra.

const $ = (s) => document.querySelector(s);
const $$ = (s) => Array.from(document.querySelectorAll(s));
const esc = (s) => String(s ?? "").replace(/[&<>"]/g, (c) =>
  ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

let App = null;
let perfil = "pessoal";
let ultimoDiagnostico = null;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function boot() {
  for (let i = 0; i < 60 && !(window.go && window.go.main && window.go.main.App); i++) await sleep(50);
  if (!(window.go && window.go.main && window.go.main.App)) {
    $("#resumo").innerHTML = '<div class="sum-main">Não foi possível falar com o motor do app.</div>';
    return;
  }
  App = window.go.main.App;
  ligarEventos();
  await diagnosticar();
}

function ligarEventos() {
  $$(".prof").forEach((b) => b.addEventListener("click", async () => {
    $$(".prof").forEach((x) => x.classList.toggle("on", x === b));
    perfil = b.dataset.perfil;
    await diagnosticar();
  }));

  $$(".tab").forEach((b) => b.addEventListener("click", () => {
    $$(".tab").forEach((x) => x.classList.toggle("on", x === b));
    $$(".view").forEach((v) => v.classList.remove("on"));
    $("#view-" + b.dataset.view).classList.add("on");
    $("#barra").style.display = b.dataset.view === "otim" ? "" : "none";
    if (b.dataset.view === "hist") carregarHistorico();
  }));

  $("#btn-recomendados").addEventListener("click", marcarRecomendados);
  $("#btn-simular").addEventListener("click", () => aplicar(true));
  $("#btn-aplicar").addEventListener("click", () => aplicar(false));
  $("#btn-desfazer-tudo").addEventListener("click", desfazerTudo);
  $("#btn-desfazer-tudo-2").addEventListener("click", desfazerTudo);
  $("#btn-medir").addEventListener("click", medir);
  $("#btn-abrir-hist").addEventListener("click", async () => {
    const erro = await App.AbrirHistorico();
    if (erro) modal("Histórico", `<p>O arquivo está em:</p><div class="term"><div>${esc(erro)}</div></div>`);
  });
  $("#modal-fechar").addEventListener("click", () => ($("#overlay").hidden = true));
}

/* ------------------------------------------------------- diagnóstico */

async function diagnosticar() {
  $("#lista").innerHTML = "";
  $("#resumo").innerHTML = '<div class="sum-main"><span class="spin"></span> Lendo o estado do seu computador…</div>';

  const d = await App.Diagnosticar(perfil);
  ultimoDiagnostico = d;

  $("#admin-area").innerHTML = d.admin
    ? '<span class="pill">administrador</span>'
    : '<span class="pill off">sem administrador</span><button class="btn" id="btn-elevar">Reabrir como admin</button>';
  const be = $("#btn-elevar");
  if (be) be.addEventListener("click", elevar);

  $("#wrap-restauracao").hidden = !d.admin;
  $("#hist-caminho").textContent = d.caminhoHistorico;

  const nota = d.recomendadosPendentes === 0
    ? "Nenhuma otimização recomendada pendente — está tudo certo por aqui."
    : `${d.recomendadosPendentes} otimização(ões) recomendada(s) ainda não aplicada(s).`;

  $("#resumo").innerHTML =
    `<div class="sum-main">${d.aplicados} de ${d.total} itens já estão como você quer.</div>
     <div class="sum-note">${esc(nota)}</div>
     <div class="grow"></div>
     <button class="btn" id="btn-rever">Diagnosticar de novo</button>`;
  $("#btn-rever").addEventListener("click", diagnosticar);

  $("#btn-desfazer-tudo").disabled = d.pendentesDesfazer === 0;
  $("#btn-desfazer-tudo").textContent =
    d.pendentesDesfazer === 0 ? "Desfazer tudo" : `Desfazer tudo (${d.pendentesDesfazer})`;

  $("#lista").innerHTML = (d.itens || []).map(item).join("");
  $$(".linkish").forEach((b) => b.addEventListener("click", () => {
    const el = b.nextElementSibling;
    el.hidden = !el.hidden;
    b.textContent = el.hidden ? "Por que isso não vem marcado?" : "Esconder explicação";
  }));
}

function item(it) {
  const aplicado = it.estado === "aplicado";
  const rotulo = { aplicado: "aplicado", nao_aplicado: "não aplicado", parcial: "parcial", desconhecido: "não foi possível ler" }[it.estado];
  const marcar = !aplicado && it.recomendado && !it.precisaAdmin;

  return `<div class="item ${aplicado ? "aplicado" : ""}">
    <input type="checkbox" data-id="${esc(it.id)}" ${aplicado ? "disabled" : ""} ${marcar ? "checked" : ""}>
    <div>
      <div class="nome">${esc(it.nome)}
        ${it.recomendado ? '<span class="tagx rec">recomendado</span>' : ""}
        ${it.precisaAdmin ? '<span class="tagx adm">precisa de admin</span>' : ""}
        <span class="tagx cat">${esc(it.categoria)} · risco ${esc(it.risco)}</span>
      </div>
      <div class="desc">${esc(it.descricao)}</div>
      <div class="detalhe">${esc(it.detalhe)}</div>
      ${it.ressalva ? `<button class="linkish">Por que isso não vem marcado?</button>
        <div class="porque" hidden><b>Ressalva honesta:</b> ${esc(it.ressalva)}</div>` : ""}
    </div>
    <span class="estado ${esc(it.estado)}">${rotulo}</span>
  </div>`;
}

function marcarRecomendados() {
  const rec = new Set((ultimoDiagnostico?.itens || []).filter((i) => i.recomendado).map((i) => i.id));
  $$('.item input[type=checkbox]').forEach((c) => { if (!c.disabled) c.checked = rec.has(c.dataset.id); });
}

function selecionados() {
  return $$('.item input[type=checkbox]:checked').map((c) => c.dataset.id);
}

/* ---------------------------------------------------------- aplicar */

async function aplicar(simular) {
  const ids = selecionados();
  if (!ids.length) {
    modal("Nada selecionado", "<p>Marque pelo menos uma otimização para continuar.</p>");
    return;
  }
  const ponto = $("#ponto-restauracao").checked && !$("#wrap-restauracao").hidden;
  travar(true);
  try {
    const res = await App.Aplicar(ids, simular, ponto);
    mostrarResultados(simular ? "Simulação — nada foi alterado" : "Pronto", res);
    if (!simular) await diagnosticar();
  } finally {
    travar(false);
  }
}

async function desfazerTudo() {
  travar(true);
  try {
    const res = await App.DesfazerTudo();
    if (!res || !res.length) {
      modal("Nada a desfazer", "<p>Este app não tem nenhuma alteração pendente nesta máquina.</p>");
      return;
    }
    mostrarResultados("Desfeito", res);
    await diagnosticar();
    if ($("#view-hist").classList.contains("on")) await carregarHistorico();
  } finally {
    travar(false);
  }
}

function travar(v) {
  $$("#barra .btn").forEach((b) => (b.disabled = v));
}

function mostrarResultados(titulo, res) {
  res = res || [];
  const linhas = res.map((r) => {
    const ic = { ok: "✓", pulado: "–", falhou: "✕" }[r.estado] || "•";
    return `<div class="res ${esc(r.estado)}">
      <span class="ic">${ic}</span>
      <span><b>${esc(r.nome)}</b><br><span class="msg">${esc(r.mensagem)}</span></span>
    </div>`;
  }).join("");
  const sair = res.some((r) => r.precisaSair && r.estado === "ok")
    ? '<div class="aviso">Alguns ajustes só aparecem depois que você sair e entrar de novo na sua conta do Windows.</div>'
    : "";
  modal(titulo, linhas + sair);
}

function modal(titulo, html) {
  $("#modal-titulo").textContent = titulo;
  $("#modal-corpo").innerHTML = html;
  $("#overlay").hidden = false;
}

async function elevar() {
  const erro = await App.ReiniciarComoAdmin();
  if (erro) { modal("Não deu certo", `<p>${esc(erro)}</p>`); return; }
  await App.Sair();
}

/* ------------------------------------------------------------- rede */

async function medir() {
  const btn = $("#btn-medir");
  btn.disabled = true;
  btn.textContent = "Medindo…";
  $("#mtu-resultado").innerHTML =
    '<div class="card"><div class="sum-main"><span class="spin"></span> Mandando pacotes de tamanhos diferentes…</div></div>';
  try {
    const m = await App.MedirMTU($("#destino").value.trim());
    $("#mtu-resultado").innerHTML = mtuHTML(m);
    const ap = $("#btn-aplicar-mtu");
    if (ap) ap.addEventListener("click", aplicarMTU);
    const det = $("#btn-detalhes");
    if (det) det.addEventListener("click", () => {
      const t = $("#tabela-passos");
      t.hidden = !t.hidden;
      det.textContent = t.hidden ? "Ver as tentativas" : "Esconder as tentativas";
    });
  } finally {
    btn.disabled = false;
    btn.textContent = "Medir de novo";
  }
}

function mtuHTML(m) {
  if (m.erro) return `<div class="card"><h2>Não deu para medir</h2><p class="muted">${esc(m.erro)}</p></div>`;

  const classe = m.veredito === "reduzir" ? "" : (m.veredito === "reduzir_opcional" ? "opcional" : "nada");
  const comandos = m.comandoOk ? `
    <p class="small muted" style="margin-top:14px">Confira você mesmo no Prompt de Comando:</p>
    <div class="term">
      <div><span class="ok">${esc(m.comandoOk)}</span>   → responde normalmente</div>
      ${m.comandoFalha ? `<div><span class="no">${esc(m.comandoFalha)}</span>   → "Pacote precisa ser fragmentado, mas o sinalizador DF está definido."</div>` : ""}
    </div>
    <p class="small muted">O número depois do <code>-l</code> é o tamanho dos dados; o MTU é esse número + 28.</p>` : "";

  const passos = (m.tentativas || []).map((p) => `<tr class="${p.passou ? "" : "falhou"}">
      <td class="mono">${p.tamanho}</td><td class="mono">${p.mtu}</td>
      <td>${esc(p.estado)}</td><td class="mono">${esc(p.comando)}</td></tr>`).join("");

  const aplicar = m.podeAplicar ? `
    <div class="inline" style="margin-top:16px">
      <button class="btn primary" id="btn-aplicar-mtu">Ajustar para ${m.sugestao}</button>
      <span class="muted small">Fica registrado no histórico e pode ser desfeito.</span>
    </div>` : "";

  return `<div class="card verdict ${classe}">
    <h2>${esc(m.resumo)}</h2>
    <p class="muted">${esc(m.explicacao)}</p>
    <p class="small muted">Adaptador <b>${esc(m.adaptador)}</b> (${esc(m.tipoAdaptador)}) · MTU configurado: ${m.mtuAtual}${m.mtuCaminho ? ` · medido no caminho: ${m.mtuCaminho}` : ""}</p>
    ${comandos}
    ${aplicar}
    ${passos ? `<button class="linkish" id="btn-detalhes">Ver as tentativas</button>
      <div id="tabela-passos" hidden><table>
        <thead><tr><th>Tamanho</th><th>MTU</th><th>Resultado</th><th>Comando equivalente</th></tr></thead>
        <tbody>${passos}</tbody></table></div>` : ""}
  </div>`;
}

async function aplicarMTU() {
  const b = $("#btn-aplicar-mtu");
  b.disabled = true;
  try {
    const res = await App.AplicarMTU(false);
    mostrarResultados("Ajuste de MTU", res);
    await medir();
  } finally {
    b.disabled = false;
  }
}

/* -------------------------------------------------------- histórico */

async function carregarHistorico() {
  const entradas = await App.Historico();
  if (!entradas || !entradas.length) {
    $("#hist-lista").innerHTML = '<div class="card"><p class="muted">Este app ainda não alterou nada nesta máquina.</p></div>';
    return;
  }
  $("#hist-lista").innerHTML = entradas.map((e) => `
    <div class="hentry ${e.ok ? "" : "erro"}">
      <span class="q">${esc(e.quando)}</span>
      <span class="a">${esc(e.acao)}</span>
      <span><b>${esc(e.item)}</b> — <span class="muted">${esc(e.resultado)}</span></span>
    </div>`).join("");
}

boot();
