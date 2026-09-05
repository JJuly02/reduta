import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useAuth } from "../auth";
import { useEventWS } from "../ws";
import { useT } from "../i18n";
import { ScoreChart } from "../ui";

export default function Scoreboard() {
  const { id } = useParams();
  const qc = useQueryClient();
  const { t } = useT();
  const { team } = useAuth();
  const sb = useQuery({ queryKey: ["sb", id], queryFn: () => api(`/events/${id}/scoreboard`) });
  const series = useQuery({ queryKey: ["sb-series", id], queryFn: () => api(`/events/${id}/scoreboard/series`) });
  useEventWS(id, (tp) => {
    if (tp === "scoreboard") {
      qc.invalidateQueries({ queryKey: ["sb", id] });
      qc.invalidateQueries({ queryKey: ["sb-series", id] });
    }
  });
  const rows = sb.data?.entries ?? [];

  return (
    <div className="container py-4">
      <div className="d-flex justify-content-between align-items-center mb-3">
        <h3 className="mb-0">{t("sb.title")}</h3>
        <span className="text-muted small">{t("sb.live")}</span>
      </div>

      <div className="card shadow-sm mb-3">
        <div className="card-body">
          <ScoreChart series={series.data?.series ?? []} />
        </div>
      </div>

      <div className="card shadow-sm">
        <table className="table table-hover mb-0 align-middle">
          <thead className="table-light">
            <tr><th style={{ width: 70 }}>{t("common.place")}</th><th>{t("common.team")}</th><th style={{ width: 90 }}>{t("common.solves")}</th><th className="text-end" style={{ width: 110 }}>{t("common.score")}</th></tr>
          </thead>
          <tbody>
            {rows.map((r: any) => {
              const mine = team && r.team_id === team.id;
              return (
                <tr key={r.team_id} className={mine ? "sb-you" : ""}>
                  <td className={r.rank === 1 ? "place-1" : ""}>{r.rank}</td>
                  <td>{r.name} {mine && <span className="badge bg-primary ms-1">{t("sb.you")}</span>}</td>
                  <td>{r.solves}</td>
                  <td className="text-end mono">{r.points}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {rows.length === 0 && <div className="text-muted p-3">{t("sb.noTeams")}</div>}
      </div>
    </div>
  );
}
