import { useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useAuth } from "../auth";
import { useT } from "../i18n";

export default function Events() {
  const { isAdmin, team } = useAuth();
  const { t } = useT();
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: ["events"], queryFn: () => api("/events") });
  const events = data?.events ?? [];
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [fb, setFb] = useState(100);
  const [err, setErr] = useState("");

  const create = useMutation({
    mutationFn: () => api("/events", { method: "POST", body: JSON.stringify({ slug, name, first_blood_bonus: Number(fb) }) }),
    onSuccess: () => { setSlug(""); setName(""); qc.invalidateQueries({ queryKey: ["events"] }); },
    onError: (e: any) => setErr(e.message),
  });

  return (
    <div className="container py-4">
      {team && (
        <div className="card shadow-sm mb-3">
          <div className="card-body d-flex flex-wrap justify-content-between align-items-center gap-2">
            <div>
              <div className="text-muted small text-uppercase">{t("team.yourTeam")}</div>
              <div className="fs-5 fw-semibold">{team.name} {team.role === "captain" && <span className="badge bg-primary ms-1">{t("team.captain")}</span>}</div>
            </div>
            {team.invite_code && (
              <div className="text-end">
                <div className="text-muted small">{t("team.inviteShare")}</div>
                <code className="fs-6">{team.invite_code}</code>
              </div>
            )}
          </div>
        </div>
      )}
      <h3 className="mb-3">{t("events.title")}</h3>
      <div className="row g-3">
        {events.map((e: any) => (
          <div className="col-sm-6 col-lg-4" key={e.id}>
            <Link to={`/events/${e.id}/play`} className="text-decoration-none">
              <div className="card h-100 shadow-sm">
                <div className="card-body d-flex justify-content-between align-items-start">
                  <div>
                    <div className="fw-semibold text-dark">{e.name}</div>
                    <div className="text-muted small mono">{e.slug}</div>
                  </div>
                  <span className={"badge " + (e.state === "running" ? "bg-success" : "bg-secondary")}>{e.state}</span>
                </div>
              </div>
            </Link>
          </div>
        ))}
        {events.length === 0 && <div className="text-muted">{t("events.none")}</div>}
      </div>

      {isAdmin && (
        <div className="card mt-4 shadow-sm">
          <div className="card-body">
            <h6 className="card-title">{t("events.newEvent")}</h6>
            <div className="row g-2">
              <div className="col-auto"><input className="form-control" placeholder={t("common.slug")} value={slug} onChange={(e) => setSlug(e.target.value)} /></div>
              <div className="col-auto"><input className="form-control" placeholder={t("common.name")} value={name} onChange={(e) => setName(e.target.value)} /></div>
              <div className="col-auto"><input className="form-control" type="number" placeholder={t("events.firstBlood")} value={fb} onChange={(e) => setFb(Number(e.target.value))} style={{ width: 140 }} /></div>
              <div className="col-auto"><button className="btn btn-primary" disabled={!slug || !name || create.isPending} onClick={() => { setErr(""); create.mutate(); }}>{t("common.create")}</button></div>
            </div>
            {err && <div className="alert alert-danger py-2 mt-2 mb-0">{err}</div>}
          </div>
        </div>
      )}
    </div>
  );
}
