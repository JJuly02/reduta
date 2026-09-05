import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../auth";
import { useT, LangSwitch } from "../i18n";

export default function Login() {
  const { user, login, register } = useAuth();
  const { t } = useT();
  const nav = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [pw, setPw] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  if (user) return <Navigate to="/" replace />;

  const submit = async (e: any) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      if (mode === "login") await login(email, pw);
      else await register(email, name, pw);
      nav("/");
    } catch (x: any) {
      setErr(x?.message || t("login.failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="container" style={{ maxWidth: 420 }}>
      <div className="d-flex justify-content-end mt-4"><LangSwitch /></div>
      <div className="card shadow-sm mt-2">
        <div className="card-body">
          <h3 className="text-center fw-bold mb-3" style={{ letterSpacing: "0.08em" }}>REDUTACTF</h3>
          <ul className="nav nav-pills nav-justified mb-3">
            <li className="nav-item"><a className={"nav-link" + (mode === "login" ? " active" : "")} role="button" onClick={() => setMode("login")}>{t("login.signin")}</a></li>
            <li className="nav-item"><a className={"nav-link" + (mode === "register" ? " active" : "")} role="button" onClick={() => setMode("register")}>{t("login.register")}</a></li>
          </ul>
          <form onSubmit={submit}>
            <div className="mb-3">
              <label className="form-label">{t("login.email")}</label>
              <input className="form-control" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            </div>
            {mode === "register" && (
              <div className="mb-3">
                <label className="form-label">{t("login.displayName")}</label>
                <input className="form-control" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
            )}
            <div className="mb-3">
              <label className="form-label">{t("login.password")}</label>
              <input className="form-control" type="password" value={pw} onChange={(e) => setPw(e.target.value)} required minLength={8} />
            </div>
            {err && <div className="alert alert-danger py-2">{err}</div>}
            <button className="btn btn-primary w-100" disabled={busy}>{mode === "login" ? t("login.signin") : t("login.createAccount")}</button>
          </form>
        </div>
      </div>
    </div>
  );
}
