import { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useEventWS } from "../ws";
import { useT } from "../i18n";

function fmtSize(n: number): string {
  if (!n && n !== 0) return "";
  const u = ["B", "KB", "MB", "GB"];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${i === 0 ? v : v.toFixed(1)} ${u[i]}`;
}

export default function Play() {
  const { id } = useParams();
  const qc = useQueryClient();
  const { t } = useT();
  const ev = useQuery({ queryKey: ["event", id], queryFn: () => api(`/events/${id}`) });
  const chalsQ = useQuery({ queryKey: ["chals", id], queryFn: () => api(`/events/${id}/challenges`) });
  const meQ = useQuery({ queryKey: ["me", id], queryFn: () => api(`/events/${id}/me`) });
  const sbQ = useQuery({ queryKey: ["sb", id], queryFn: () => api(`/events/${id}/scoreboard`) });

  useEventWS(id, (t2) => {
    if (t2 === "challenges.changed" || t2 === "scoreboard") {
      qc.invalidateQueries({ queryKey: ["chals", id] });
      qc.invalidateQueries({ queryKey: ["me", id] });
      qc.invalidateQueries({ queryKey: ["sb", id] });
    }
  });

  const [sel, setSel] = useState<any>(null);
  const [search, setSearch] = useState("");
  const [cat, setCat] = useState<string>("");
  const [hideSolved, setHideSolved] = useState(false);

  const chals = chalsQ.data?.challenges ?? [];
  const solved = new Set<string>(meQ.data?.solved ?? []);
  const hasTeam = !!meQ.data?.team_id;

  const allCats = useMemo(() => {
    const s = new Set<string>();
    for (const c of chals) s.add(c.category || "misc");
    return [...s].sort();
  }, [chals]);

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return chals.filter((c: any) => {
      if (cat && (c.category || "misc") !== cat) return false;
      if (hideSolved && solved.has(c.id)) return false;
      if (q && !c.title.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [chals, search, cat, hideSolved, solved]);

  const cats: Record<string, any[]> = {};
  for (const c of visible) (cats[c.category || "misc"] ||= []).push(c);
  const catNames = Object.keys(cats).sort();

  // rank from scoreboard
  const rows = sbQ.data?.entries ?? [];
  const myRow = rows.find((r: any) => r.team_id === meQ.data?.team_id);
  const rank = myRow ? myRow.rank : null;
  const totalTeams = rows.length;
  const solvedCount = solved.size;
  const totalChals = chals.length;
  const pct = totalChals > 0 ? Math.round((solvedCount / totalChals) * 100) : 0;

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["chals", id] });
    qc.invalidateQueries({ queryKey: ["me", id] });
    qc.invalidateQueries({ queryKey: ["sb", id] });
  };

  return (
    <div className="container py-4">
      <div className="d-flex justify-content-between align-items-center mb-3">
        <h3 className="mb-0">{ev.data?.name ?? t("play.event")}</h3>
        {ev.data?.state && <span className={"badge " + (ev.data.state === "running" ? "bg-success" : "bg-secondary")}>{ev.data.state}</span>}
      </div>

      {!hasTeam && <div className="alert alert-warning">{t("play.notRegistered")}</div>}

      <div className="progress-strip mb-2">
        <div className="cell">
          <div className="k">{t("play.rank")}</div>
          <div className="v">{rank ? `#${rank}` : "-"} {totalTeams > 0 && <small>/ {totalTeams}</small>}</div>
        </div>
        <div className="cell">
          <div className="k">{t("play.points")}</div>
          <div className="v mono">{meQ.data?.points ?? 0}</div>
        </div>
        <div className="cell">
          <div className="k">{t("play.solved")}</div>
          <div className="v">{solvedCount} <small>/ {totalChals}</small></div>
          <div className="strip-bar-track"><div className="strip-bar" style={{ width: pct + "%" }} /></div>
        </div>
        <div className="cell">
          <div className="k">{t("play.categories")}</div>
          <div className="v">{allCats.length}</div>
        </div>
      </div>

      <div className="board-toolbar">
        <input className="form-control" style={{ maxWidth: 280 }} placeholder={t("play.searchPlaceholder")} value={search} onChange={(e) => setSearch(e.target.value)} />
        <button className={"chip" + (cat === "" ? " active" : "")} onClick={() => setCat("")}>{t("common.all")}</button>
        {allCats.map((c) => (
          <button key={c} className={"chip" + (cat === c ? " active" : "")} onClick={() => setCat(c)}>{c}</button>
        ))}
        <label className="ms-auto d-inline-flex align-items-center gap-2 small text-muted" style={{ cursor: "pointer" }}>
          <input type="checkbox" className="form-check-input mt-0" checked={hideSolved} onChange={(e) => setHideSolved(e.target.checked)} />
          {t("play.hideSolved")}
        </label>
      </div>

      {chals.length === 0 && <div className="text-muted py-4">{t("play.noChallenges")}</div>}
      {chals.length > 0 && visible.length === 0 && <div className="text-muted py-4">{t("play.noMatch")}</div>}

      {catNames.map((c) => (
        <div key={c}>
          <div className="category-title">{c} <span className="count">{cats[c].filter((x) => solved.has(x.id)).length} / {cats[c].length}</span></div>
          <div className="row g-3">
            {cats[c].map((ch: any) => (
              <div className="col-6 col-md-4 col-lg-3" key={ch.id}>
                <Card c={ch} solved={solved.has(ch.id)} onClick={() => !ch.locked && setSel(ch)} />
              </div>
            ))}
          </div>
        </div>
      ))}

      {sel && (
        <ChallengePanel
          eventId={id!}
          c={sel}
          solved={solved.has(sel.id)}
          hasTeam={hasTeam}
          onClose={() => setSel(null)}
          onSolved={refresh}
        />
      )}
    </div>
  );
}

function Card({ c, solved, onClick }: { c: any; solved: boolean; onClick: () => void }) {
  const { t } = useT();
  const cls = "chal-card" + (solved ? " solved" : "") + (c.locked ? " locked" : "");
  const tags: string[] = c.tags ?? [];
  return (
    <div className={cls} onClick={onClick}>
      <div className="top">
        <div className="name">{c.title}</div>
        <div className="value">{c.points}</div>
      </div>
      {tags.length > 0 && <div className="tags">{tags.slice(0, 3).map((tg) => <span className="tag" key={tg}>{tg}</span>)}</div>}
      <div className="foot">
        <span className="solves">{c.solves ?? 0} {t("common.solves")}</span>
        {solved ? <span className="status status-solved">{t("play.solvedBadge")}</span>
          : c.locked ? <span className="status status-locked">{c.status === "scheduled" ? t("play.scheduled") : t("play.locked")}</span>
          : null}
      </div>
    </div>
  );
}

function ChallengePanel({ eventId, c, solved, hasTeam, onClose, onSolved }: any) {
  const { t } = useT();
  const [tab, setTab] = useState<"desc" | "attempts">("desc");
  const detail = useQuery({ queryKey: ["chal", c.id], queryFn: () => api(`/events/${eventId}/challenges/${c.id}`) });
  const attempts = useQuery({ queryKey: ["chal-att", c.id], queryFn: () => api(`/events/${eventId}/challenges/${c.id}/attempts`), enabled: tab === "attempts" });
  const [flag, setFlag] = useState("");
  const [msg, setMsg] = useState<{ ok?: boolean; text: string } | null>(null);

  const solveText = (c.solves ?? 0) === 0 ? t("play.solvedByNone") : (c.solves === 1 ? t("play.solvedByOne") : t("play.solvedByN", { n: c.solves }));

  const submit = async () => {
    setMsg(null);
    try {
      const r: any = await api(`/events/${eventId}/challenges/${c.id}/submit`, { method: "POST", body: JSON.stringify({ flag }) });
      if (r.correct) {
        setMsg({ ok: true, text: r.first_blood ? t("play.firstBloodPlus", { n: r.points }) : t("play.correctPlus", { n: r.points }) });
        setFlag(""); onSolved();
      } else setMsg({ ok: false, text: t("play.incorrect") });
    } catch (e: any) { setMsg({ ok: false, text: e.message }); }
  };

  return (
    <div className="modal d-block" tabIndex={-1} onClick={onClose}>
      <div className="modal-dialog modal-dialog-centered modal-lg" onClick={(e) => e.stopPropagation()}>
        <div className="modal-content">
          <div className="modal-header">
            <div>
              <h5 className="modal-title mb-1">{c.title} <small className="mono" style={{ color: "var(--accent-600)" }}>{c.points}</small></h5>
              <div className="d-flex gap-2 align-items-center">
                <span className="badge bg-secondary text-uppercase">{c.category}</span>
                {solved && <span className="badge bg-success">{t("play.solvedBadge")}</span>}
                <span className="small text-muted">{solveText}</span>
              </div>
            </div>
            <button type="button" className="btn-close" onClick={onClose}></button>
          </div>
          <div className="modal-body">
            <div className="chal-tabs">
              <button className={tab === "desc" ? "active" : ""} onClick={() => setTab("desc")}>{t("play.tabDescription")}</button>
              <button className={tab === "attempts" ? "active" : ""} onClick={() => setTab("attempts")}>{t("play.tabAttempts")}</button>
            </div>

            {tab === "desc" && (
              <>
                <p style={{ whiteSpace: "pre-wrap" }}>{detail.data?.description_md || t("play.noDescription")}</p>
                {(detail.data?.files ?? []).length > 0 && (
                  <div className="chal-files mt-3">
                    <div className="small text-uppercase text-muted mb-1">{t("play.files")}</div>
                    <ul className="list-unstyled mb-0">
                      {detail.data.files.map((f: any) => (
                        <li key={f.id} className="d-flex align-items-center gap-2">
                          <a href={`/api/v1/events/${eventId}/challenges/${c.id}/files/${f.id}`} download={f.name}>{f.name}</a>
                          <span className="text-muted small mono">{fmtSize(f.size)}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </>
            )}
            {tab === "attempts" && (
              <div className="mb-2">
                {(attempts.data?.attempts ?? []).length === 0 && <div className="text-muted small">{t("play.noAttempts")}</div>}
                {(attempts.data?.attempts ?? []).map((a: any, i: number) => (
                  <div className="attempt-row" key={i}>
                    <span className={a.correct ? "status-solved" : "text-muted"}>{a.correct ? t("play.attemptCorrect") : t("play.attemptWrong")}</span>
                    <span className="text-muted">{new Date(a.submitted_at).toLocaleString()}</span>
                  </div>
                ))}
              </div>
            )}

            {!hasTeam && <div className="alert alert-warning py-2 mt-3">{t("play.joinToSubmit")}</div>}
            <div className="input-group mt-3">
              <input className="form-control flag-input" placeholder={t("play.flagPlaceholder")} value={flag} onChange={(e) => setFlag(e.target.value)} onKeyDown={(e) => e.key === "Enter" && flag && hasTeam && submit()} />
              <button className="btn btn-primary" disabled={!flag || !hasTeam} onClick={submit}>{t("common.submit")}</button>
            </div>
            {msg && <div className={"alert py-2 mt-3 mb-0 " + (msg.ok ? "alert-success" : "alert-danger")}>{msg.text}</div>}
          </div>
        </div>
      </div>
    </div>
  );
}
