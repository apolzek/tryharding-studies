import React from "react";
import { Routes, Route, Link, Navigate } from "react-router-dom";
import Home from "./pages/Home.jsx";
import ChallengeList from "./pages/ChallengeList.jsx";
import ChallengePage from "./pages/ChallengePage.jsx";
import AdminLogin from "./pages/AdminLogin.jsx";
import AdminDashboard from "./pages/AdminDashboard.jsx";
import AdminChallengeEdit from "./pages/AdminChallengeEdit.jsx";

function RequireAdmin({ children }) {
  const token = localStorage.getItem("sre_token");
  if (!token) return <Navigate to="/admin/login" replace />;
  return children;
}

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          <span className="logo">⚙︎</span>
          <span>SRE Challenges</span>
        </Link>
        <nav>
          <Link to="/challenges">Challenges</Link>
          <Link to="/admin">Admin</Link>
        </nav>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/challenges" element={<ChallengeList />} />
          <Route path="/challenges/:slug" element={<ChallengePage />} />
          <Route path="/admin/login" element={<AdminLogin />} />
          <Route
            path="/admin"
            element={
              <RequireAdmin>
                <AdminDashboard />
              </RequireAdmin>
            }
          />
          <Route
            path="/admin/challenges/new"
            element={
              <RequireAdmin>
                <AdminChallengeEdit />
              </RequireAdmin>
            }
          />
          <Route
            path="/admin/challenges/:id"
            element={
              <RequireAdmin>
                <AdminChallengeEdit />
              </RequireAdmin>
            }
          />
        </Routes>
      </main>
    </div>
  );
}
