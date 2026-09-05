import { NavLink, Navigate, Route, Routes, useParams } from "react-router-dom";
import { useAuth } from "./auth";
import { useT, LangSwitch } from "./i18n";
import { NotificationToaster } from "./ui";
import Login from "./pages/Login";
import Events from "./pages/Events";
import Play from "./pages/Play";
import Scoreboard from "./pages/Scoreboard";
import Admin from "./pages/Admin";
import Notifications from "./pages/Notifications";
import TeamSetup from "./pages/TeamSetup";

function Wrench() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="currentColor" viewBox="0 0 16 16" style={{ verticalAlign: "-2px", marginRight: 4 }}>
      <path d="M.102 2.223A3.004 3.004 0 0 0 3.78 5.897l6.341 6.252A3.003 3.003 0 0 0 13 16a3 3 0 1 0-.851-5.878L5.897 3.781A3.004 3.004 0 0 0 2.223.102l2.141 2.142L4 4l-1.757.364zm13.37 9.019.528.026.287.445.445.287.026.529L15 13l-.242.471-.026.529-.445.287-.287.445-.529.026L13 15l-.471-.242-.529-.026-.287-.445-.445-.287-.026-.529L11 13l.242-.471.026-.529.445-.287.287-.445.529-.026L13 11z" />
    </svg>
  );
}

function Nav() {
  const { user, logout, isAdmin } = useAuth();
  const { t } = useT();
  const params = useParams();
  const eid = (params as any).id as string | undefined;
  const link = ({ isActive }: { isActive: boolean }) => "nav-link" + (isActive ? " active" : "");
  return (
    <nav className="navbar navbar-expand navbar-dark bg-dark px-3">
      <NavLink to="/" className="navbar-brand">RedutaCTF</NavLink>
      <ul className="navbar-nav me-auto">
        {!eid && <li className="nav-item"><NavLink to="/" end className={link}>{t("nav.events")}</NavLink></li>}
        {eid && <li className="nav-item"><NavLink to={`/events/${eid}/play`} className={link}>{t("nav.challenges")}</NavLink></li>}
        {eid && <li className="nav-item"><NavLink to={`/events/${eid}/scoreboard`} className={link}>{t("nav.scoreboard")}</NavLink></li>}
        {eid && <li className="nav-item"><NavLink to={`/events/${eid}/notifications`} className={link}>{t("nav.notifications")}</NavLink></li>}
      </ul>
      <ul className="navbar-nav align-items-center gap-2">
        {isAdmin && eid && (
          <li className="nav-item">
            <NavLink to={`/events/${eid}/admin`} className={link} title={t("nav.adminPanel")}><Wrench />{t("nav.adminPanel")}</NavLink>
          </li>
        )}
        {user && <li className="nav-item"><span className="navbar-text text-light small mx-1">{user.display_name}</span></li>}
        <li className="nav-item"><LangSwitch dark /></li>
        {user && <li className="nav-item"><button className="btn btn-outline-light btn-sm" onClick={() => logout()}>{t("nav.logout")}</button></li>}
      </ul>
      <NotificationToaster eventId={eid} />
    </nav>
  );
}

function RequireAuth({ children }: { children: any }) {
  const { user, loading } = useAuth();
  const { t } = useT();
  if (loading) return <div className="container py-5 text-muted">{t("common.loading")}</div>;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function RequireTeam({ children }: { children: any }) {
  const { user, team, isAdmin, loading } = useAuth();
  const { t } = useT();
  if (loading) return <div className="container py-5 text-muted">{t("common.loading")}</div>;
  if (!user) return <Navigate to="/login" replace />;
  if (!team && !isAdmin) return <Navigate to="/onboarding" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/onboarding" element={<RequireAuth><TeamSetup /></RequireAuth>} />
      <Route path="/" element={<RequireAuth><RequireTeam><><Nav /><Events /></></RequireTeam></RequireAuth>} />
      <Route path="/events/:id/play" element={<RequireAuth><RequireTeam><><Nav /><Play /></></RequireTeam></RequireAuth>} />
      <Route path="/events/:id/scoreboard" element={<RequireAuth><RequireTeam><><Nav /><Scoreboard /></></RequireTeam></RequireAuth>} />
      <Route path="/events/:id/notifications" element={<RequireAuth><RequireTeam><><Nav /><Notifications /></></RequireTeam></RequireAuth>} />
      <Route path="/events/:id/admin" element={<RequireAuth><Admin /></RequireAuth>} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
