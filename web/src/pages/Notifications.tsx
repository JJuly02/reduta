import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useEventWS } from "../ws";
import { useT } from "../i18n";

export default function Notifications() {
  const { id } = useParams();
  const qc = useQueryClient();
  const { t } = useT();
  const q = useQuery({ queryKey: ["notifs", id], queryFn: () => api(`/events/${id}/notifications`) });
  useEventWS(id, (tp) => { if (tp === "notification") qc.invalidateQueries({ queryKey: ["notifs", id] }); });
  const items = q.data?.notifications ?? [];
  return (
    <div className="container py-4" style={{ maxWidth: 760 }}>
      <h3 className="mb-3">{t("notif.title")}</h3>
      {items.length === 0 && <div className="text-muted">{t("notif.none")}</div>}
      {items.map((n: any) => (
        <div className="card shadow-sm mb-2" key={n.id}>
          <div className="card-body">
            <div className="d-flex justify-content-between">
              <strong>{n.title}</strong>
              <span className="text-muted small">{new Date(n.created_at).toLocaleString()}</span>
            </div>
            {n.content && <div className="mt-1" style={{ whiteSpace: "pre-wrap" }}>{n.content}</div>}
          </div>
        </div>
      ))}
    </div>
  );
}
