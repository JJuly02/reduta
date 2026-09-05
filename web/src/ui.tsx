import { useState } from "react";
import { useEventWS } from "./ws";
import { useT } from "./i18n";

// ScoreChart draws a CTFd-style stepped multi-line progress chart in plain SVG.
export function ScoreChart({ series }: { series: any[] }) {
  const { t } = useT();
  const W = 940, H = 300, padL = 46, padR = 14, padT = 12, padB = 28;
  const colors = ["#4f46e5", "#dc3545", "#16a34a", "#fd7e14", "#6f42c1", "#0dcaf0", "#d63384", "#0d6efd", "#ca8a04", "#6610f2"];
  const times: number[] = [];
  let maxS = 1;
  for (const s of series) for (const p of s.data) {
    times.push(new Date(p.t).getTime());
    if (p.score > maxS) maxS = p.score;
  }
  if (times.length === 0) return <div className="text-muted p-3">{t("sb.noHistory")}</div>;
  const minT = Math.min(...times);
  const maxT = Math.max(...times);
  const x = (tt: number) => padL + ((tt - minT) / ((maxT - minT) || 1)) * (W - padL - padR);
  const y = (v: number) => H - padB - (v / maxS) * (H - padT - padB);

  return (
    <div>
      <svg viewBox={`0 0 ${W} ${H}`} style={{ width: "100%", height: "auto" }} role="img" aria-label="Score progression">
        {[0, 0.25, 0.5, 0.75, 1].map((f, i) => {
          const yy = y(maxS * f);
          return (
            <g key={i}>
              <line x1={padL} y1={yy} x2={W - padR} y2={yy} stroke="#e5e7eb" />
              <text x={padL - 6} y={yy + 3} textAnchor="end" fontSize="10" fill="#adb5bd">{Math.round(maxS * f)}</text>
            </g>
          );
        })}
        {series.map((s, i) => {
          const c = colors[i % colors.length];
          const pts: string[] = [];
          let prevY = 0;
          s.data.forEach((p: any, idx: number) => {
            const px = x(new Date(p.t).getTime());
            const py = y(p.score);
            if (idx > 0) pts.push(`${px},${prevY}`);
            pts.push(`${px},${py}`);
            prevY = py;
          });
          const last = s.data[s.data.length - 1];
          return (
            <g key={s.team_id}>
              <polyline points={pts.join(" ")} fill="none" stroke={c} strokeWidth="2" />
              {last && <circle cx={x(new Date(last.t).getTime())} cy={y(last.score)} r="3" fill={c} />}
            </g>
          );
        })}
      </svg>
      <div className="d-flex flex-wrap gap-3 mt-2">
        {series.map((s, i) => (
          <span key={s.team_id} className="small d-inline-flex align-items-center">
            <span style={{ width: 10, height: 10, background: colors[i % colors.length], display: "inline-block", borderRadius: 2, marginRight: 6 }}></span>
            {s.name} <span className="text-muted ms-1">({s.points})</span>
          </span>
        ))}
      </div>
    </div>
  );
}

// NotificationToaster shows live admin notifications as CTFd-style toasts.
export function NotificationToaster({ eventId }: { eventId?: string }) {
  const [toasts, setToasts] = useState<{ id: number; title: string; content: string }[]>([]);
  useEventWS(eventId, (type, data) => {
    if (type === "notification") {
      const t = { id: Date.now() + Math.random(), title: data?.title || "Notification", content: data?.content || "" };
      setToasts((cur) => [...cur, t]);
      setTimeout(() => setToasts((cur) => cur.filter((x) => x.id !== t.id)), 8000);
    }
  });
  if (toasts.length === 0) return null;
  return (
    <div className="toast-container position-fixed top-0 end-0 p-3" style={{ zIndex: 1080 }}>
      {toasts.map((t) => (
        <div key={t.id} className="toast show mb-2" role="alert">
          <div className="toast-header">
            <strong className="me-auto">{t.title}</strong>
            <button type="button" className="btn-close" onClick={() => setToasts((cur) => cur.filter((x) => x.id !== t.id))}></button>
          </div>
          {t.content && <div className="toast-body">{t.content}</div>}
        </div>
      ))}
    </div>
  );
}
