import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useAuth } from "../auth";
import { useT, LangSwitch } from "../i18n";

const SECTIONS = ["statistics", "notifications", "events", "challenges", "blocks", "library", "teams", "submissions", "scoreboard"] as const;
type Section = (typeof SECTIONS)[number];

export default function Admin() {
  const { id } = useParams();
  const eid = id!;
  const { logout } = useAuth();
  const { t } = useT();
  const [tab, setTab] = useState<Section>("statistics");
  return (
    <div style={{ background: "#fff", minHeight: "100vh" }}>
      <nav className="navbar navbar-expand navbar-dark bg-dark px-3">
        <span className="navbar-brand">RedutaCTF</span>
        <ul className="navbar-nav me-auto flex-wrap">
          {SECTIONS.map((sec) => (
            <li className="nav-item" key={sec}>
              <a className={"nav-link" + (tab === sec ? " active" : "")} role="button" onClick={() => setTab(sec)}>{t("admin." + sec)}</a>
            </li>
          ))}
        </ul>
        <div className="me-2"><LangSwitch dark /></div>
        <Link className="btn btn-outline-light btn-sm me-2" to={`/events/${eid}/play`}>{t("nav.viewSite")}</Link>
        <button className="btn btn-outline-light btn-sm" onClick={() => logout()}>{t("nav.logout")}</button>
      </nav>

      <div className="bg-light border-bottom py-4">
        <h1 className="text-center m-0" style={{ fontWeight: 300 }}>{t("admin." + tab)}</h1>
      </div>

      <div className="container py-4">
        {tab === "statistics" && <StatsTab eid={eid} />}
        {tab === "notifications" && <NotificationsTab eid={eid} />}
        {tab === "events" && <EventsTab eid={eid} />}
        {tab === "challenges" && <ChallengesTab eid={eid} />}
        {tab === "blocks" && <BlocksTab eid={eid} />}
        {tab === "library" && <LibraryTab eid={eid} />}
        {tab === "teams" && <TeamsTab eid={eid} />}
        {tab === "submissions" && <SubmissionsTab eid={eid} />}
        {tab === "scoreboard" && <AdminScoreboard eid={eid} />}
      </div>
    </div>
  );
}

const CATCOLORS = ["#4f46e5", "#16a34a", "#fd7e14", "#0dcaf0", "#d63384", "#6610f2", "#ca8a04", "#dc3545", "#0d6efd", "#20c997"];

function StatsTab({ eid }: { eid: string }) {
  const { t } = useT();
  const q = useQuery({ queryKey: ["stats", eid], queryFn: () => api(`/events/${eid}/stats`) });
  const chalsQ = useQuery({ queryKey: ["admin-chals", eid], queryFn: () => api(`/events/${eid}/challenges`) });
  const s = q.data;
  const chals = chalsQ.data?.challenges ?? [];
  const byCat: Record<string, number> = {};
  for (const c of chals) byCat[c.category || "misc"] = (byCat[c.category || "misc"] || 0) + 1;
  const cats = Object.entries(byCat).sort((a, b) => b[1] - a[1]);
  const total = chals.length || 1;
  const cell = (n: any, label: string) => (
    <div className="cell"><div className="n">{n ?? "-"}</div><div className="l">{label}</div></div>
  );
  return (
    <div>
      <div className="stat-strip mb-4">
        {cell(s?.teams, t("st.teams"))}
        {cell(s?.challenges, t("st.challenges"))}
        {cell(s?.published, t("st.published"))}
        {cell(s?.solves, t("st.solves"))}
        {cell(s?.submissions, t("st.submissions"))}
      </div>
      {cats.length > 0 && (
        <div className="card shadow-sm"><div className="card-body">
          <h6 className="text-muted">{t("st.byCategory")}</h6>
          <div className="catbar mb-3">
            {cats.map(([name, n], i) => (
              <span key={name} title={`${name}: ${n}`} style={{ width: (n / total) * 100 + "%", background: CATCOLORS[i % CATCOLORS.length] }} />
            ))}
          </div>
          <div className="d-flex flex-wrap gap-3">
            {cats.map(([name, n], i) => (
              <span key={name} className="small d-inline-flex align-items-center">
                <span style={{ width: 10, height: 10, borderRadius: 2, background: CATCOLORS[i % CATCOLORS.length], display: "inline-block", marginRight: 6 }} />
                {name} <span className="text-muted ms-1">({n})</span>
              </span>
            ))}
          </div>
        </div></div>
      )}
    </div>
  );
}

function NotificationsTab({ eid }: { eid: string }) {
  const qc = useQueryClient();
  const list = useQuery({ queryKey: ["admin-notifs", eid], queryFn: () => api(`/events/${eid}/notifications`) });
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [note, setNote] = useState("");
  const send = async () => {
    setNote("");
    try {
      await api(`/events/${eid}/notifications`, { method: "POST", body: JSON.stringify({ title, content }) });
      setTitle(""); setContent(""); setNote("Notification sent to all players.");
      qc.invalidateQueries({ queryKey: ["admin-notifs", eid] });
    } catch (e: any) { setNote("error: " + e.message); }
  };
  const items = list.data?.notifications ?? [];
  return (
    <div className="row">
      <div className="col-lg-7">
        <div className="mb-3">
          <label className="form-label fw-semibold">Title</label>
          <input className="form-control" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Notification title" />
        </div>
        <div className="mb-3">
          <label className="form-label fw-semibold">Content</label>
          <textarea className="form-control" rows={4} value={content} onChange={(e) => setContent(e.target.value)} placeholder="Notification contents" />
        </div>
        <button className="btn btn-success" disabled={!title} onClick={send}>Submit</button>
        {note && <div className="alert alert-info py-2 mt-3">{note}</div>}
      </div>
      <div className="col-lg-5">
        <h6 className="text-muted">Recent</h6>
        {items.length === 0 && <div className="text-muted small">Nothing sent yet.</div>}
        {items.map((n: any) => (
          <div className="border-bottom py-2" key={n.id}>
            <div className="fw-semibold">{n.title}</div>
            <div className="small text-muted">{new Date(n.created_at).toLocaleString()}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

function EventsTab({ eid }: { eid: string }) {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["events"], queryFn: () => api("/events") });
  const events = q.data?.events ?? [];
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [fb, setFb] = useState(100);
  const [err, setErr] = useState("");
  const create = useMutation({
    mutationFn: () => api("/events", { method: "POST", body: JSON.stringify({ slug, name, first_blood_bonus: Number(fb) }) }),
    onSuccess: () => { setSlug(""); setName(""); qc.invalidateQueries({ queryKey: ["events"] }); },
    onError: (e: any) => setErr(e.message),
  });
  const setState = async (id: string, state: string) => {
    await api(`/events/${id}/state`, { method: "PATCH", body: JSON.stringify({ state }) });
    qc.invalidateQueries({ queryKey: ["events"] });
  };
  return (
    <div>
      <div className="card shadow-sm mb-3">
        <div className="card-body">
          <h6 className="card-title">New event</h6>
          <div className="row g-2">
            <div className="col-auto"><input className="form-control" placeholder="slug" value={slug} onChange={(e) => setSlug(e.target.value)} /></div>
            <div className="col-auto"><input className="form-control" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} /></div>
            <div className="col-auto"><input className="form-control" type="number" placeholder="first blood" value={fb} onChange={(e) => setFb(Number(e.target.value))} style={{ width: 140 }} /></div>
            <div className="col-auto"><button className="btn btn-primary" disabled={!slug || !name || create.isPending} onClick={() => { setErr(""); create.mutate(); }}>Create</button></div>
          </div>
          {err && <div className="alert alert-danger py-2 mt-2 mb-0">{err}</div>}
        </div>
      </div>
      <div className="card shadow-sm">
        <table className="table table-hover mb-0 align-middle">
          <thead className="table-light"><tr><th>Name</th><th>Slug</th><th>State</th><th style={{ width: 240 }}>Set state</th></tr></thead>
          <tbody>
            {events.map((e: any) => (
              <tr key={e.id}>
                <td>{e.name} {e.id === eid && <span className="badge bg-primary ms-1">current</span>}</td>
                <td className="mono text-muted">{e.slug}</td>
                <td><span className={"badge " + (e.state === "running" ? "bg-success" : "bg-secondary")}>{e.state}</span></td>
                <td className="btn-group btn-group-sm">
                  {["draft", "running", "ended"].map((st) => (
                    <button key={st} className={"btn " + (e.state === st ? "btn-primary" : "btn-outline-secondary")} onClick={() => setState(e.id, st)}>{st}</button>
                  ))}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function TeamsTab({ eid }: { eid: string }) {
  const { t } = useT();
  const qc = useQueryClient();
  const all = useQuery({ queryKey: ["org-teams"], queryFn: () => api(`/teams`) });
  const assigned = useQuery({ queryKey: ["event-teams", eid], queryFn: () => api(`/events/${eid}/teams`) });
  const teams = all.data?.teams ?? [];
  const assignedSet = new Set<string>((assigned.data?.teams ?? []).map((t: any) => t.id));
  const assign = async (teamId: string) => {
    await api(`/events/${eid}/event-teams`, { method: "POST", body: JSON.stringify({ team_id: teamId }) });
    qc.invalidateQueries({ queryKey: ["event-teams", eid] });
  };
  const unassign = async (teamId: string) => {
    await api(`/events/${eid}/event-teams/${teamId}`, { method: "DELETE" });
    qc.invalidateQueries({ queryKey: ["event-teams", eid] });
  };
  return (
    <div>
      <p className="text-muted">Assign teams to this event. Only assigned teams can play it and appear on its scoreboard.</p>
      <div className="card shadow-sm">
        <table className="table table-hover mb-0 align-middle">
          <thead className="table-light"><tr><th>{t("common.team")}</th><th style={{ width: 150 }}>In this event</th><th style={{ width: 130 }}>{t("common.actions")}</th></tr></thead>
          <tbody>
            {teams.map((tm: any) => {
              const isIn = assignedSet.has(tm.id);
              return (
                <tr key={tm.id}>
                  <td>{tm.name}</td>
                  <td>{isIn ? <span className="badge bg-success">assigned</span> : <span className="badge bg-secondary">not assigned</span>}</td>
                  <td>{isIn
                    ? <button className="btn btn-sm btn-outline-danger" onClick={() => unassign(tm.id)}>Remove</button>
                    : <button className="btn btn-sm btn-outline-success" onClick={() => assign(tm.id)}>Add</button>}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {teams.length === 0 && <div className="text-muted p-3">No teams have registered yet.</div>}
      </div>
    </div>
  );
}

function SubmissionsTab({ eid }: { eid: string }) {
  const q = useQuery({ queryKey: ["admin-subs", eid], queryFn: () => api(`/events/${eid}/submissions`) });
  const subs = q.data?.submissions ?? [];
  return (
    <div className="card shadow-sm">
      <div className="table-responsive">
        <table className="table table-hover table-sm mb-0 align-middle">
          <thead className="table-light"><tr><th>ID</th><th>User</th><th>Team</th><th>Challenge</th><th>Type</th><th>Date</th></tr></thead>
          <tbody>
            {subs.map((r: any) => (
              <tr key={r.id}>
                <td className="text-muted">{r.id}</td>
                <td>{r.user}</td>
                <td>{r.team}</td>
                <td>{r.challenge}</td>
                <td>{r.correct ? <span className="badge bg-success">correct</span> : <span className="badge bg-secondary">incorrect</span>}</td>
                <td className="small text-muted">{new Date(r.submitted_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {subs.length === 0 && <div className="text-muted p-3">No submissions yet.</div>}
    </div>
  );
}

function AdminScoreboard({ eid }: { eid: string }) {
  const { t } = useT();
  const q = useQuery({ queryKey: ["admin-sb", eid], queryFn: () => api(`/events/${eid}/scoreboard`) });
  const rows = q.data?.entries ?? [];
  return (
    <div className="card shadow-sm">
      <table className="table table-hover mb-0 align-middle">
        <thead className="table-light"><tr><th style={{ width: 70 }}>{t("common.place")}</th><th>{t("common.team")}</th><th>{t("common.solves")}</th><th className="text-end">{t("common.score")}</th></tr></thead>
        <tbody>{rows.map((r: any) => (<tr key={r.team_id}><td>{r.rank}</td><td>{r.name}</td><td>{r.solves}</td><td className="text-end mono">{r.points}</td></tr>))}</tbody>
      </table>
      {rows.length === 0 && <div className="text-muted p-3">{t("sb.noTeams")}</div>}
    </div>
  );
}

function EventStateControl({ eid }: { eid: string }) {
  const qc = useQueryClient();
  const ev = useQuery({ queryKey: ["event", eid], queryFn: () => api(`/events/${eid}`) });
  const cur = ev.data?.state;
  const set = async (state: string) => {
    await api(`/events/${eid}/state`, { method: "PATCH", body: JSON.stringify({ state }) });
    qc.invalidateQueries({ queryKey: ["event", eid] });
  };
  return (
    <div className="btn-group">
      {["draft", "running", "ended"].map((s) => (
        <button key={s} className={"btn btn-sm " + (cur === s ? "btn-primary" : "btn-outline-secondary")} onClick={() => set(s)}>{s}</button>
      ))}
    </div>
  );
}

function ChallengesTab({ eid }: { eid: string }) {
  const { t } = useT();
  const qc = useQueryClient();
  const chalsQ = useQuery({ queryKey: ["admin-chals", eid], queryFn: () => api(`/events/${eid}/challenges`) });
  const blocksQ = useQuery({ queryKey: ["blocks", eid], queryFn: () => api(`/events/${eid}/blocks`) });
  const chals = chalsQ.data?.challenges ?? [];
  const blocks = blocksQ.data?.blocks ?? [];
  const bmap: Record<string, string> = {};
  for (const b of blocks) bmap[b.id] = b.name;

  const [search, setSearch] = useState("");
  const [catF, setCatF] = useState("");
  const [stateF, setStateF] = useState("");
  const [sortKey, setSortKey] = useState<"title" | "category" | "state" | "points" | "solves">("title");
  const [sortDir, setSortDir] = useState<1 | -1>(1);
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [matchAll, setMatchAll] = useState(false);
  const [assignBlock, setAssignBlock] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [note, setNote] = useState("");
  const [lastJob, setLastJob] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showIO, setShowIO] = useState(false);
  const [confirmDel, setConfirmDel] = useState(false);
  const [delText, setDelText] = useState("");

  const categories = useMemo(() => [...new Set(chals.map((c: any) => c.category || "misc"))].sort(), [chals]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    const out = chals.filter((c: any) => {
      if (catF && (c.category || "misc") !== catF) return false;
      if (stateF && c.state !== stateF) return false;
      if (q && !c.title.toLowerCase().includes(q)) return false;
      return true;
    });
    out.sort((a: any, b: any) => {
      let av = a[sortKey], bv = b[sortKey];
      if (sortKey === "points" || sortKey === "solves") { av = av ?? 0; bv = bv ?? 0; return (av - bv) * sortDir; }
      return String(av ?? "").localeCompare(String(bv ?? "")) * sortDir;
    });
    return out;
  }, [chals, search, catF, stateF, sortKey, sortDir]);

  const canMatchAll = search.trim() === "";
  const allFilteredSelected = filtered.length > 0 && filtered.every((c: any) => sel.has(c.id));
  const selectionCount = matchAll ? filtered.length : sel.size;

  const sort = (k: typeof sortKey) => {
    if (sortKey === k) setSortDir((d) => (d === 1 ? -1 : 1));
    else { setSortKey(k); setSortDir(1); }
  };
  const arrow = (k: typeof sortKey) => sortKey === k ? <span className="arrow">{sortDir === 1 ? "↑" : "↓"}</span> : null;

  const toggle = (id: string) => {
    setMatchAll(false);
    setSel((cur) => { const n = new Set(cur); n.has(id) ? n.delete(id) : n.add(id); return n; });
  };
  const toggleAll = () => {
    setMatchAll(false);
    if (allFilteredSelected) setSel(new Set());
    else setSel(new Set(filtered.map((c: any) => c.id)));
  };
  const clearSel = () => { setSel(new Set()); setMatchAll(false); };

  const buildFilter = () => {
    const f: any = {};
    if (catF) f.category = catF;
    if (stateF) f.state = stateF;
    return f;
  };
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["admin-chals", eid] });
    qc.invalidateQueries({ queryKey: ["stats", eid] });
    clearSel();
  };
  const setState = async (id: string, state: string) => {
    await api(`/events/${eid}/challenges/${id}/state`, { method: "PATCH", body: JSON.stringify({ state }) });
    qc.invalidateQueries({ queryKey: ["admin-chals", eid] });
    qc.invalidateQueries({ queryKey: ["stats", eid] });
  };
  const bulk = async (type: string, params: any = {}) => {
    setNote("");
    const selector = matchAll
      ? { mode: "filter", filter: buildFilter() }
      : { mode: "ids", ids: [...sel] };
    const r: any = await api(`/events/${eid}/challenges/bulk`, { method: "POST", body: JSON.stringify({ selector, action: { type, params } }) });
    setNote(t("ac.affected", { action: type, n: r.affected }));
    setLastJob(type === "delete" ? null : r.job_id);
    refresh();
  };
  const undo = async () => {
    if (!lastJob) return;
    const r: any = await api(`/bulk-jobs/${lastJob}/undo`, { method: "POST" });
    setNote(t("ac.restored", { n: r.restored }));
    setLastJob(null);
    qc.invalidateQueries({ queryKey: ["admin-chals", eid] });
    qc.invalidateQueries({ queryKey: ["stats", eid] });
  };
  const doDelete = async () => { setConfirmDel(false); setDelText(""); await bulk("delete"); };

  const stateBadge = (s: string) => "badge state-badge " + (s === "published" ? "bg-success" : s === "hidden" ? "bg-secondary" : "bg-warning text-dark");

  return (
    <div>
      <div className="admin-toolbar">
        <input className="form-control grow" placeholder={t("ac.searchName")} value={search} onChange={(e) => setSearch(e.target.value)} style={{ maxWidth: 260 }} />
        <select className="form-select" style={{ width: "auto" }} value={catF} onChange={(e) => setCatF(e.target.value)}>
          <option value="">{t("ac.filterCategory")}</option>
          {categories.map((c) => <option key={c as string} value={c as string}>{c as string}</option>)}
        </select>
        <select className="form-select" style={{ width: "auto" }} value={stateF} onChange={(e) => setStateF(e.target.value)}>
          <option value="">{t("ac.filterState")}</option>
          <option value="draft">{t("ac.draft")}</option>
          <option value="published">{t("ac.published")}</option>
          <option value="hidden">{t("ac.hidden")}</option>
        </select>
        <div className="ms-auto d-flex gap-2">
          <EventStateControl eid={eid} />
          <button className="btn btn-sm btn-outline-secondary" onClick={() => setShowIO((v) => !v)}>{t("ac.importExport")}</button>
          <button className="btn btn-sm btn-primary" onClick={() => setShowCreate(true)}>{t("ac.newChallenge")}</button>
        </div>
      </div>

      {showIO && <ImportExport eid={eid} onDone={() => qc.invalidateQueries({ queryKey: ["admin-chals", eid] })} />}

      {allFilteredSelected && !matchAll && canMatchAll && (
        <div className="match-banner">
          <span>{t("ac.selected", { n: sel.size })}</span>
          <button className="btn btn-sm btn-link p-0" onClick={() => setMatchAll(true)}>{t("ac.selectAllMatching", { n: filtered.length })}</button>
        </div>
      )}
      {matchAll && (
        <div className="match-banner">
          <span>{t("ac.matchingSelected", { n: filtered.length })}</span>
          <button className="btn btn-sm btn-link p-0" onClick={clearSel}>{t("ac.clear")}</button>
        </div>
      )}

      <div className="card shadow-sm">
        <div className="table-responsive">
          <table className="table table-hover table-sm mb-0 align-middle">
            <thead className="table-light">
              <tr>
                <th style={{ width: 34 }}><input type="checkbox" className="form-check-input" checked={allFilteredSelected} onChange={toggleAll} /></th>
                <th className="sortable" onClick={() => sort("title")}>{t("common.name")}{arrow("title")}</th>
                <th className="sortable" onClick={() => sort("category")}>{t("common.category")}{arrow("category")}</th>
                <th>{t("ac.block")}</th>
                <th className="sortable" onClick={() => sort("state")}>{t("common.state")}{arrow("state")}</th>
                <th className="sortable text-end" onClick={() => sort("solves")}>{t("common.solves")}{arrow("solves")}</th>
                <th className="sortable text-end" onClick={() => sort("points")}>{t("common.value")}{arrow("points")}</th>
                <th style={{ width: 120 }}>{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((c: any) => (
                <tr key={c.id} className={sel.has(c.id) || matchAll ? "table-active" : ""}>
                  <td><input type="checkbox" className="form-check-input" checked={sel.has(c.id) || matchAll} onChange={() => toggle(c.id)} /></td>
                  <td>{c.title}</td>
                  <td className="text-muted">{c.category}</td>
                  <td className="text-muted">{c.block_id ? bmap[c.block_id] || "?" : "-"}</td>
                  <td><span className={stateBadge(c.state)}>{c.state}</span></td>
                  <td className="text-end mono">{c.solves ?? 0}</td>
                  <td className="text-end mono">{c.points}</td>
                  <td>{c.state !== "published"
                    ? <button className="btn btn-sm btn-outline-success" onClick={() => setState(c.id, "published")}>{t("ac.publish")}</button>
                    : <button className="btn btn-sm btn-outline-secondary" onClick={() => setState(c.id, "hidden")}>{t("ac.hide")}</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {chals.length === 0 && <div className="text-muted p-3">{t("ac.none")}</div>}
      </div>

      {note && <div className="alert alert-info py-2 mt-3 d-flex justify-content-between align-items-center">
        <span>{note}</span>
        {lastJob && <button className="btn btn-sm btn-outline-primary" onClick={undo}>{t("ac.undo")}</button>}
      </div>}

      {selectionCount > 0 && (
        <div className="bulk-bar">
          <strong>{t("ac.selected", { n: selectionCount })}</strong>
          <button className="btn btn-sm btn-outline-success" onClick={() => bulk("publish")}>{t("ac.publish")}</button>
          <button className="btn btn-sm btn-outline-secondary" onClick={() => bulk("hide")}>{t("ac.hide")}</button>
          <div className="input-group input-group-sm" style={{ width: 240 }}>
            <select className="form-select" value={assignBlock} onChange={(e) => setAssignBlock(e.target.value)}>
              <option value="">{t("ac.assignBlock")}</option>
              {blocks.map((b: any) => <option key={b.id} value={b.id}>{b.name}</option>)}
            </select>
            <button className="btn btn-outline-primary" disabled={!assignBlock} onClick={() => bulk("assign_block", { block_id: assignBlock })}>{t("ac.assign")}</button>
          </div>
          <div className="input-group input-group-sm" style={{ width: 200 }}>
            <input className="form-control" placeholder={t("ac.addTag")} value={tagInput} onChange={(e) => setTagInput(e.target.value)} />
            <button className="btn btn-outline-primary" disabled={!tagInput.trim()} onClick={() => { bulk("add_tags", { tags: [tagInput.trim()] }); setTagInput(""); }}>{t("ac.add")}</button>
          </div>
          <button className="btn btn-sm btn-outline-danger" onClick={() => { setDelText(""); setConfirmDel(true); }}>{t("ac.delete")}</button>
          <button className="btn btn-sm btn-link ms-auto" onClick={clearSel}>{t("ac.clear")}</button>
        </div>
      )}

      {confirmDel && (
        <div className="modal d-block" tabIndex={-1} onClick={() => setConfirmDel(false)}>
          <div className="modal-dialog modal-dialog-centered" onClick={(e) => e.stopPropagation()}>
            <div className="modal-content">
              <div className="modal-header"><h5 className="modal-title">{t("ac.confirmDeleteTitle")}</h5>
                <button className="btn-close" onClick={() => setConfirmDel(false)}></button></div>
              <div className="modal-body">
                <p>{t("ac.confirmDeleteBody", { n: selectionCount })}</p>
                <input className="form-control" value={delText} onChange={(e) => setDelText(e.target.value)} placeholder={String(selectionCount)} />
              </div>
              <div className="modal-footer">
                <button className="btn btn-outline-secondary" onClick={() => setConfirmDel(false)}>{t("common.cancel")}</button>
                <button className="btn btn-danger" disabled={delText !== String(selectionCount)} onClick={doDelete}>{t("ac.confirmDeleteBtn", { n: selectionCount })}</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {showCreate && (
        <>
          <div className="drawer-scrim" onClick={() => setShowCreate(false)} />
          <div className="drawer">
            <div className="drawer-head">
              <h5 className="m-0">{t("ac.newChallenge")}</h5>
              <button className="btn-close" onClick={() => setShowCreate(false)}></button>
            </div>
            <div className="drawer-body">
              <CreateChallenge eid={eid} onDone={() => { qc.invalidateQueries({ queryKey: ["admin-chals", eid] }); qc.invalidateQueries({ queryKey: ["stats", eid] }); setShowCreate(false); }} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function CreateChallenge({ eid, onDone }: { eid: string; onDone: () => void }) {
  const { t } = useT();
  const [title, setTitle] = useState("");
  const [category, setCategory] = useState("misc");
  const [points, setPoints] = useState(100);
  const [desc, setDesc] = useState("");
  const [flag, setFlag] = useState("");
  const [state, setState] = useState("published");
  const [err, setErr] = useState("");
  const create = async () => {
    setErr("");
    try { await api(`/events/${eid}/challenges`, { method: "POST", body: JSON.stringify({ title, category, description_md: desc, state, scoring: { type: "static", points: Number(points) }, flags: [{ value: flag }] }) }); onDone(); }
    catch (e: any) { setErr(e.message); }
  };
  return (
    <div>
      <div className="mb-2"><label className="form-label">{t("common.name")}</label><input className="form-control" value={title} onChange={(e) => setTitle(e.target.value)} /></div>
      <div className="row g-2 mb-2">
        <div className="col-6"><label className="form-label">{t("common.category")}</label><input className="form-control" value={category} onChange={(e) => setCategory(e.target.value)} /></div>
        <div className="col-6"><label className="form-label">{t("common.value")}</label><input className="form-control" type="number" value={points} onChange={(e) => setPoints(Number(e.target.value))} /></div>
      </div>
      <div className="mb-2"><label className="form-label">{t("play.tabDescription")}</label><textarea className="form-control" rows={4} value={desc} onChange={(e) => setDesc(e.target.value)} /></div>
      <div className="mb-2"><label className="form-label">Flag</label><input className="form-control mono" placeholder={t("ac.createFlag")} value={flag} onChange={(e) => setFlag(e.target.value)} /></div>
      <div className="mb-3"><label className="form-label">{t("common.state")}</label>
        <select className="form-select" value={state} onChange={(e) => setState(e.target.value)}>
          <option value="draft">{t("ac.draft")}</option><option value="published">{t("ac.published")}</option><option value="hidden">{t("ac.hidden")}</option>
        </select>
      </div>
      <button className="btn btn-primary w-100" disabled={!title || !flag} onClick={create}>{t("common.create")}</button>
      {err && <div className="alert alert-danger py-2 mt-2 mb-0">{err}</div>}
    </div>
  );
}

function ImportExport({ eid, onDone }: { eid: string; onDone: () => void }) {
  const [text, setText] = useState("");
  const [format, setFormat] = useState("native");
  const [dry, setDry] = useState(true);
  const [note, setNote] = useState("");
  const run = async () => {
    setNote("");
    try {
      const q = `?format=${format}${dry ? "&dry_run=true" : ""}`;
      const r: any = await api(`/events/${eid}/import${q}`, { method: "POST", body: text });
      setNote(`plan: created ${r.plan.created}, updated ${r.plan.updated}` + (r.dry_run ? " (dry run)" : ""));
      if (!r.dry_run) onDone();
    } catch (e: any) { setNote("error: " + e.message); }
  };
  return (
    <div className="card shadow-sm mb-3"><div className="card-body">
      <div className="d-flex justify-content-between align-items-center">
        <h6 className="card-title mb-0">Import / export</h6>
        <a className="btn btn-sm btn-outline-secondary" href={`/api/v1/events/${eid}/export`} target="_blank" rel="noreferrer">Export JSON</a>
      </div>
      <div className="mt-3">
        <div className="row g-2 align-items-center mb-2">
          <div className="col-auto"><select className="form-select form-select-sm" value={format} onChange={(e) => setFormat(e.target.value)}><option value="native">native JSON</option><option value="ctfd">CTFd export</option></select></div>
          <div className="col-auto form-check"><input className="form-check-input" type="checkbox" id="dry" checked={dry} onChange={(e) => setDry(e.target.checked)} /><label className="form-check-label" htmlFor="dry">dry run</label></div>
          <div className="col-auto"><button className="btn btn-sm btn-primary" disabled={!text.trim()} onClick={run}>Run import</button></div>
        </div>
        <textarea className="form-control mono" rows={6} placeholder='{"challenges":[...]}' value={text} onChange={(e) => setText(e.target.value)} />
      </div>
      {note && <div className="alert alert-info py-2 mt-2 mb-0">{note}</div>}
    </div></div>
  );
}

function BlocksTab({ eid }: { eid: string }) {
  const qc = useQueryClient();
  const blocksQ = useQuery({ queryKey: ["blocks", eid], queryFn: () => api(`/events/${eid}/blocks`) });
  const blocks = blocksQ.data?.blocks ?? [];
  const [name, setName] = useState("");
  const [pos, setPos] = useState(0);
  const [err, setErr] = useState("");
  const create = async () => {
    setErr("");
    try { await api(`/events/${eid}/blocks`, { method: "POST", body: JSON.stringify({ name, position: Number(pos) }) }); setName(""); qc.invalidateQueries({ queryKey: ["blocks", eid] }); }
    catch (e: any) { setErr(e.message); }
  };
  return (
    <div>
      <div className="card shadow-sm mb-3"><div className="card-body">
        <h6 className="card-title">New block</h6>
        <div className="row g-2">
          <div className="col-auto"><input className="form-control" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} /></div>
          <div className="col-auto"><input className="form-control" type="number" placeholder="position" value={pos} onChange={(e) => setPos(Number(e.target.value))} style={{ width: 130 }} /></div>
          <div className="col-auto"><button className="btn btn-primary" disabled={!name} onClick={create}>Create</button></div>
        </div>
        {err && <div className="alert alert-danger py-2 mt-2 mb-0">{err}</div>}
      </div></div>
      <div className="row g-3">
        {blocks.map((b: any) => (<div className="col-sm-6 col-lg-4" key={b.id}><div className="card shadow-sm"><div className="card-body d-flex justify-content-between"><strong>{b.name}</strong><span className="text-muted mono">#{b.position}</span></div></div></div>))}
        {blocks.length === 0 && <div className="text-muted">No blocks yet.</div>}
      </div>
    </div>
  );
}

function LibraryTab({ eid }: { eid: string }) {
  const qc = useQueryClient();
  const libQ = useQuery({ queryKey: ["library"], queryFn: () => api(`/challenges`) });
  const lib = libQ.data?.challenges ?? [];
  const [slug, setSlug] = useState("");
  const [title, setTitle] = useState("");
  const [category, setCategory] = useState("misc");
  const [points, setPoints] = useState(100);
  const [flag, setFlag] = useState("");
  const [note, setNote] = useState("");
  const create = async () => {
    setNote("");
    try { await api(`/challenges`, { method: "POST", body: JSON.stringify({ slug, title, category, scoring: { type: "static", points: Number(points) }, flags: [{ value: flag }] }) }); setSlug(""); setTitle(""); setFlag(""); qc.invalidateQueries({ queryKey: ["library"] }); }
    catch (e: any) { setNote("error: " + e.message); }
  };
  const embed = async (cid: string) => {
    setNote("");
    try { await api(`/events/${eid}/challenges/from-library`, { method: "POST", body: JSON.stringify({ challenge_id: cid }) }); setNote("Embedded into this event as a draft challenge."); qc.invalidateQueries({ queryKey: ["admin-chals", eid] }); }
    catch (e: any) { setNote("error: " + e.message); }
  };
  const clone = async (cid: string, curSlug: string) => {
    setNote("");
    try { await api(`/challenges/${cid}/clone`, { method: "POST", body: JSON.stringify({ slug: curSlug + "-copy-" + Date.now().toString(36) }) }); qc.invalidateQueries({ queryKey: ["library"] }); }
    catch (e: any) { setNote("error: " + e.message); }
  };
  return (
    <div>
      <div className="card shadow-sm mb-3"><div className="card-body">
        <h6 className="card-title">New library challenge</h6>
        <div className="row g-2">
          <div className="col-md-2"><input className="form-control" placeholder="slug" value={slug} onChange={(e) => setSlug(e.target.value)} /></div>
          <div className="col-md-3"><input className="form-control" placeholder="title" value={title} onChange={(e) => setTitle(e.target.value)} /></div>
          <div className="col-md-2"><input className="form-control" placeholder="category" value={category} onChange={(e) => setCategory(e.target.value)} /></div>
          <div className="col-md-2"><input className="form-control" type="number" value={points} onChange={(e) => setPoints(Number(e.target.value))} /></div>
          <div className="col-md-3"><input className="form-control mono" placeholder="flag{...}" value={flag} onChange={(e) => setFlag(e.target.value)} /></div>
        </div>
        <button className="btn btn-primary mt-2" disabled={!slug || !title || !flag} onClick={create}>Add</button>
      </div></div>
      {note && <div className="alert alert-info py-2">{note}</div>}
      <div className="card shadow-sm"><div className="table-responsive">
        <table className="table table-hover table-sm mb-0 align-middle">
          <thead className="table-light"><tr><th>Title</th><th>Slug</th><th>Category</th><th>Rev</th><th style={{ width: 190 }}>Actions</th></tr></thead>
          <tbody>{lib.map((c: any) => (<tr key={c.id}><td>{c.title}</td><td className="mono text-muted">{c.slug}</td><td className="text-muted">{c.category}</td><td className="mono">{c.current_rev}</td><td><button className="btn btn-sm btn-outline-primary me-2" onClick={() => embed(c.id)}>Embed</button><button className="btn btn-sm btn-outline-secondary" onClick={() => clone(c.id, c.slug)}>Clone</button></td></tr>))}</tbody>
        </table>
      </div>{lib.length === 0 && <div className="text-muted p-3">Library is empty.</div>}</div>
    </div>
  );
}
