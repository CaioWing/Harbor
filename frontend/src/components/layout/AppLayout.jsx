import { NavLink } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";

const navItems = [
  { to: "/", label: "Dashboard" },
  { to: "/devices", label: "Devices" },
  { to: "/deployments", label: "Deployments" },
  { to: "/audit", label: "Audit" }
];

function activeClass({ isActive }) {
  return isActive ? "sidebar__link sidebar__link--active" : "sidebar__link";
}

export default function AppLayout({ title, subtitle, actions, children }) {
  const { signOut } = useAuth();

  return (
    <div className="layout">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">Harbor Frontend</p>
          <h1 className="sidebar__title">Control Center</h1>
        </div>

        <nav className="sidebar__nav">
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.to === "/"} className={activeClass}>
              {item.label}
            </NavLink>
          ))}
        </nav>

        <button className="ghost sidebar__logout" onClick={signOut}>
          Sair
        </button>
      </aside>

      <main className="content">
        <header className="content__header">
          <div>
            <p className="eyebrow">Operacao</p>
            <h2 className="content__title">{title}</h2>
            {subtitle ? <p className="subtitle">{subtitle}</p> : null}
          </div>
          {actions ? <div className="content__actions">{actions}</div> : null}
        </header>

        {children}
      </main>
    </div>
  );
}
