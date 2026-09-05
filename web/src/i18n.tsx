import { createContext, useContext, useState, ReactNode, useCallback } from "react";

export type Lang = "en" | "pl";

type Dict = Record<string, string>;

const en: Dict = {
  // chrome / nav
  "nav.events": "Events",
  "nav.challenges": "Challenges",
  "nav.scoreboard": "Scoreboard",
  "nav.notifications": "Notifications",
  "nav.adminPanel": "Admin Panel",
  "nav.viewSite": "View Site",
  "nav.logout": "Log out",
  "common.loading": "Loading...",
  "common.points": "points",
  "common.solves": "solves",
  "common.score": "Score",
  "common.place": "Place",
  "common.team": "Team",
  "common.category": "Category",
  "common.state": "State",
  "common.value": "Value",
  "common.actions": "Actions",
  "common.submit": "Submit",
  "common.cancel": "Cancel",
  "common.close": "Close",
  "common.create": "Create",
  "common.name": "Name",
  "common.slug": "Slug",
  "common.all": "All",
  "common.search": "Search",
  // login
  "login.signin": "Sign in",
  "login.register": "Register",
  "login.email": "Email",
  "login.displayName": "Display name",
  "login.password": "Password",
  "login.createAccount": "Create account",
  "login.failed": "Request failed",
  // onboarding / team setup
  "team.setup": "Set up your team",
  "team.intro": "You need a team to take part. Create one and invite your teammates, or join an existing team with its invite code.",
  "team.create": "Create a team",
  "team.joinCode": "Join with code",
  "team.teamName": "Team name",
  "team.teamNamePlaceholder": "Your team name",
  "team.inviteCode": "Invite code",
  "team.inviteCodePlaceholder": "paste invite code",
  "team.createBtn": "Create team",
  "team.joinBtn": "Join team",
  "team.yourTeam": "Your team",
  "team.captain": "captain",
  "team.inviteShare": "Invite code (share with teammates)",
  // events
  "events.title": "Events",
  "events.none": "No events yet.",
  "events.newEvent": "New event",
  "events.firstBlood": "first blood",
  "events.running": "running",
  // play
  "play.event": "Event",
  "play.notRegistered": "Your team is not registered for this event yet. Ask an admin to add your team, then refresh.",
  "play.rank": "Rank",
  "play.points": "Points",
  "play.solved": "Solved",
  "play.categories": "Categories",
  "play.searchPlaceholder": "Search challenges...",
  "play.hideSolved": "Hide solved",
  "play.noChallenges": "No published challenges yet.",
  "play.noMatch": "No challenges match your filters.",
  "play.solvedBadge": "solved",
  "play.locked": "locked",
  "play.scheduled": "scheduled",
  "play.solvedByOne": "solved by 1 team",
  "play.solvedByN": "solved by {n} teams",
  "play.solvedByNone": "not solved yet",
  "play.tabDescription": "Description",
  "play.tabAttempts": "Your attempts",
  "play.noDescription": "No description provided.",
  "play.files": "Files",
  "play.noAttempts": "Your team has not submitted anything yet.",
  "play.attemptCorrect": "correct",
  "play.attemptWrong": "incorrect",
  "play.joinToSubmit": "Join a team to submit flags.",
  "play.flagPlaceholder": "flag{...}",
  "play.correctPlus": "Correct, plus {n} points.",
  "play.firstBloodPlus": "Correct. First blood, plus {n} points.",
  "play.incorrect": "Incorrect flag.",
  // scoreboard
  "sb.title": "Scoreboard",
  "sb.live": "live",
  "sb.noTeams": "No teams yet.",
  "sb.noHistory": "No score history yet.",
  "sb.you": "you",
  // notifications page
  "notif.title": "Notifications",
  "notif.none": "No notifications yet.",
  // admin sections
  "admin.statistics": "Statistics",
  "admin.notifications": "Notifications",
  "admin.events": "Events",
  "admin.challenges": "Challenges",
  "admin.blocks": "Blocks",
  "admin.library": "Library",
  "admin.teams": "Teams",
  "admin.submissions": "Submissions",
  "admin.scoreboard": "Scoreboard",
  // admin challenges toolbar / bulk
  "ac.newChallenge": "New challenge",
  "ac.importExport": "Import / export",
  "ac.filterCategory": "All categories",
  "ac.filterState": "All states",
  "ac.searchName": "Search by name...",
  "ac.block": "Block",
  "ac.noneBlock": "no block",
  "ac.publish": "Publish",
  "ac.hide": "Hide",
  "ac.assignBlock": "assign to block...",
  "ac.assign": "Assign",
  "ac.addTag": "add tag",
  "ac.add": "Add",
  "ac.delete": "Delete",
  "ac.clear": "Clear",
  "ac.undo": "Undo",
  "ac.selected": "{n} selected",
  "ac.selectAllMatching": "Select all {n} challenges matching this filter",
  "ac.matchingSelected": "All {n} challenges matching this filter are selected.",
  "ac.affected": "{action}: {n} affected",
  "ac.restored": "Undo done: {n} restored",
  "ac.none": "No challenges. Create one or embed from the library.",
  "ac.confirmDeleteTitle": "Delete challenges",
  "ac.confirmDeleteBody": "This permanently deletes {n} challenges and cannot be undone. Type {n} to confirm.",
  "ac.confirmDeleteBtn": "Delete {n}",
  "ac.createName": "name",
  "ac.createFlag": "flag{...}",
  "ac.draft": "draft",
  "ac.published": "published",
  "ac.hidden": "hidden",
  // admin stats
  "st.teams": "teams",
  "st.challenges": "challenges",
  "st.published": "published",
  "st.solves": "solves",
  "st.submissions": "submissions",
  "st.byCategory": "By category",
};

const pl: Dict = {
  "nav.events": "Wydarzenia",
  "nav.challenges": "Zadania",
  "nav.scoreboard": "Ranking",
  "nav.notifications": "Powiadomienia",
  "nav.adminPanel": "Panel admina",
  "nav.viewSite": "Podgląd strony",
  "nav.logout": "Wyloguj",
  "common.loading": "Ładowanie...",
  "common.points": "punktów",
  "common.solves": "rozwiązań",
  "common.score": "Wynik",
  "common.place": "Miejsce",
  "common.team": "Drużyna",
  "common.category": "Kategoria",
  "common.state": "Stan",
  "common.value": "Wartość",
  "common.actions": "Akcje",
  "common.submit": "Wyślij",
  "common.cancel": "Anuluj",
  "common.close": "Zamknij",
  "common.create": "Utwórz",
  "common.name": "Nazwa",
  "common.slug": "Slug",
  "common.all": "Wszystkie",
  "common.search": "Szukaj",
  "login.signin": "Zaloguj się",
  "login.register": "Rejestracja",
  "login.email": "Email",
  "login.displayName": "Nazwa wyświetlana",
  "login.password": "Hasło",
  "login.createAccount": "Załóż konto",
  "login.failed": "Żądanie nie powiodło się",
  "team.setup": "Skonfiguruj drużynę",
  "team.intro": "Aby wziąć udział, potrzebujesz drużyny. Załóż własną i zaproś innych, albo dołącz do istniejącej za pomocą kodu zaproszenia.",
  "team.create": "Załóż drużynę",
  "team.joinCode": "Dołącz kodem",
  "team.teamName": "Nazwa drużyny",
  "team.teamNamePlaceholder": "Nazwa Twojej drużyny",
  "team.inviteCode": "Kod zaproszenia",
  "team.inviteCodePlaceholder": "wklej kod zaproszenia",
  "team.createBtn": "Załóż drużynę",
  "team.joinBtn": "Dołącz do drużyny",
  "team.yourTeam": "Twoja drużyna",
  "team.captain": "kapitan",
  "team.inviteShare": "Kod zaproszenia (podaj go drużynie)",
  "events.title": "Wydarzenia",
  "events.none": "Brak wydarzeń.",
  "events.newEvent": "Nowe wydarzenie",
  "events.firstBlood": "first blood",
  "events.running": "trwa",
  "play.event": "Wydarzenie",
  "play.notRegistered": "Twoja drużyna nie jest jeszcze zapisana na to wydarzenie. Poproś administratora o dodanie drużyny i odśwież.",
  "play.rank": "Miejsce",
  "play.points": "Punkty",
  "play.solved": "Rozwiązane",
  "play.categories": "Kategorie",
  "play.searchPlaceholder": "Szukaj zadań...",
  "play.hideSolved": "Ukryj rozwiązane",
  "play.noChallenges": "Brak opublikowanych zadań.",
  "play.noMatch": "Żadne zadanie nie pasuje do filtrów.",
  "play.solvedBadge": "rozwiązane",
  "play.locked": "zablokowane",
  "play.scheduled": "zaplanowane",
  "play.solvedByOne": "rozwiązane przez 1 drużynę",
  "play.solvedByN": "rozwiązane przez {n} drużyn",
  "play.solvedByNone": "jeszcze nierozwiązane",
  "play.tabDescription": "Opis",
  "play.tabAttempts": "Twoje próby",
  "play.noDescription": "Brak opisu.",
  "play.files": "Pliki",
  "play.noAttempts": "Twoja drużyna nie wysłała jeszcze żadnej próby.",
  "play.attemptCorrect": "poprawna",
  "play.attemptWrong": "błędna",
  "play.joinToSubmit": "Dołącz do drużyny, aby wysyłać flagi.",
  "play.flagPlaceholder": "flaga{...}",
  "play.correctPlus": "Poprawnie, +{n} punktów.",
  "play.firstBloodPlus": "Poprawnie. First blood, +{n} punktów.",
  "play.incorrect": "Błędna flaga.",
  "sb.title": "Ranking",
  "sb.live": "na żywo",
  "sb.noTeams": "Brak drużyn.",
  "sb.noHistory": "Brak historii wyników.",
  "sb.you": "Ty",
  "notif.title": "Powiadomienia",
  "notif.none": "Brak powiadomień.",
  "admin.statistics": "Statystyki",
  "admin.notifications": "Powiadomienia",
  "admin.events": "Wydarzenia",
  "admin.challenges": "Zadania",
  "admin.blocks": "Bloki",
  "admin.library": "Biblioteka",
  "admin.teams": "Drużyny",
  "admin.submissions": "Zgłoszenia",
  "admin.scoreboard": "Ranking",
  "ac.newChallenge": "Nowe zadanie",
  "ac.importExport": "Import / eksport",
  "ac.filterCategory": "Wszystkie kategorie",
  "ac.filterState": "Wszystkie stany",
  "ac.searchName": "Szukaj po nazwie...",
  "ac.block": "Blok",
  "ac.noneBlock": "brak bloku",
  "ac.publish": "Opublikuj",
  "ac.hide": "Ukryj",
  "ac.assignBlock": "przypisz do bloku...",
  "ac.assign": "Przypisz",
  "ac.addTag": "dodaj tag",
  "ac.add": "Dodaj",
  "ac.delete": "Usuń",
  "ac.clear": "Wyczyść",
  "ac.undo": "Cofnij",
  "ac.selected": "zaznaczono: {n}",
  "ac.selectAllMatching": "Zaznacz wszystkie {n} zadań pasujących do filtra",
  "ac.matchingSelected": "Zaznaczono wszystkie {n} zadań pasujących do filtra.",
  "ac.affected": "{action}: zmieniono {n}",
  "ac.restored": "Cofnięto: przywrócono {n}",
  "ac.none": "Brak zadań. Utwórz nowe lub osadź z biblioteki.",
  "ac.confirmDeleteTitle": "Usuń zadania",
  "ac.confirmDeleteBody": "To trwale usunie {n} zadań i nie można tego cofnąć. Wpisz {n}, aby potwierdzić.",
  "ac.confirmDeleteBtn": "Usuń {n}",
  "ac.createName": "nazwa",
  "ac.createFlag": "flaga{...}",
  "ac.draft": "szkic",
  "ac.published": "opublikowane",
  "ac.hidden": "ukryte",
  "st.teams": "drużyny",
  "st.challenges": "zadania",
  "st.published": "opublikowane",
  "st.solves": "rozwiązania",
  "st.submissions": "zgłoszenia",
  "st.byCategory": "Wg kategorii",
};

const DICTS: Record<Lang, Dict> = { en, pl };

type I18nCtx = {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
};

const Ctx = createContext<I18nCtx>(null!);
export const useT = () => useContext(Ctx);

function initialLang(): Lang {
  try {
    const s = localStorage.getItem("reduta.lang");
    if (s === "pl" || s === "en") return s;
  } catch { /* ignore */ }
  return "en";
}

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang);
  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    try { localStorage.setItem("reduta.lang", l); } catch { /* ignore */ }
  }, []);
  const t = useCallback((key: string, vars?: Record<string, string | number>) => {
    let s = DICTS[lang][key] ?? DICTS.en[key] ?? key;
    if (vars) for (const k of Object.keys(vars)) s = s.replace(new RegExp("\\{" + k + "\\}", "g"), String(vars[k]));
    return s;
  }, [lang]);
  return <Ctx.Provider value={{ lang, setLang, t }}>{children}</Ctx.Provider>;
}

// LangSwitch is a compact EN | PL toggle for the navbar and auth screens.
export function LangSwitch({ dark }: { dark?: boolean }) {
  const { lang, setLang } = useT();
  const base = dark ? "lang-switch lang-switch-dark" : "lang-switch";
  return (
    <div className={base} role="group" aria-label="Language">
      <button type="button" className={lang === "en" ? "active" : ""} onClick={() => setLang("en")}>EN</button>
      <button type="button" className={lang === "pl" ? "active" : ""} onClick={() => setLang("pl")}>PL</button>
    </div>
  );
}
