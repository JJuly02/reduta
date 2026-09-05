import { useState } from "react";
import { Navigate } from "react-router-dom";
import { api } from "../api";
import { useAuth } from "../auth";
import { useT, LangSwitch } from "../i18n";

export default function TeamSetup() {
  const { user, team, isAdmin, refreshTeam } = useAuth();
  const { t } = useT();
  const [mode, setMode] = useState<"create" | "join">("create");
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  if (!user) return <Navigate to="/login" replace />;
  if (team || isAdmin) return <Navigate to="/" replace />;

  const submit = async () => {
    setErr(""); setBusy(true);
    try {
      if (mode === "create") await api("/teams", { method: "POST", body: JSON.stringify({ name }) });
      else await api("/teams/join", { method: "POST", body: JSON.stringify({ invite_code: code }) });
      await refreshTeam();
    } catch (e: any) { setErr(e.message); } finally { setBusy(false); }
  };

  return (
    <div className="container" style={{ maxWidth: 480 }}>
      <div className="d-flex justify-content-end mt-4"><LangSwitch /></div>
      <div className="card shadow-sm mt-2">
        <div className="card-body">
          <h4 className="mb-1">{t("team.setup")}</h4>
          <p className="text-muted small">{t("team.intro")}</p>
          <ul className="nav nav-pills nav-justified mb-3">
            <li className="nav-item"><a className={"nav-link" + (mode === "create" ? " active" : "")} role="button" onClick={() => setMode("create")}>{t("team.create")}</a></li>
            <li className="nav-item"><a className={"nav-link" + (mode === "join" ? " active" : "")} role="button" onClick={() => setMode("join")}>{t("team.joinCode")}</a></li>
          </ul>
          {mode === "create" ? (
            <div className="mb-3">
              <label className="form-label">{t("team.teamName")}</label>
              <input className="form-control" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("team.teamNamePlaceholder")} />
            </div>
          ) : (
            <div className="mb-3">
              <label className="form-label">{t("team.inviteCode")}</label>
              <input className="form-control mono" value={code} onChange={(e) => setCode(e.target.value)} placeholder={t("team.inviteCodePlaceholder")} />
            </div>
          )}
          {err && <div className="alert alert-danger py-2">{err}</div>}
          <button className="btn btn-primary w-100" disabled={busy || (mode === "create" ? !name : !code)} onClick={submit}>
            {mode === "create" ? t("team.createBtn") : t("team.joinBtn")}
          </button>
        </div>
      </div>
    </div>
  );
}
