import React, { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api.js";

const DIFF_ORDER = { easy: 0, medium: 1, hard: 2 };

export default function ChallengeList() {
  const [challenges, setChallenges] = useState([]);
  const [error, setError] = useState(null);
  const [category, setCategory] = useState("all");
  const [difficulty, setDifficulty] = useState("all");

  useEffect(() => {
    api.listChallenges().then(setChallenges).catch((e) => setError(e.message));
  }, []);

  const categories = useMemo(
    () => ["all", ...Array.from(new Set(challenges.map((c) => c.category)))],
    [challenges]
  );

  const filtered = challenges
    .filter((c) => category === "all" || c.category === category)
    .filter((c) => difficulty === "all" || c.difficulty === difficulty)
    .sort(
      (a, b) =>
        a.category.localeCompare(b.category) ||
        (DIFF_ORDER[a.difficulty] ?? 9) - (DIFF_ORDER[b.difficulty] ?? 9) ||
        a.title.localeCompare(b.title)
    );

  if (error) return <div className="error">Error: {error}</div>;

  return (
    <div className="list-page">
      <div className="filters">
        <label>
          Category:
          <select value={category} onChange={(e) => setCategory(e.target.value)}>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>
        <label>
          Difficulty:
          <select value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
            <option value="all">all</option>
            <option value="easy">easy</option>
            <option value="medium">medium</option>
            <option value="hard">hard</option>
          </select>
        </label>
        <span className="count">{filtered.length} challenges</span>
      </div>

      <div className="grid">
        {filtered.map((c) => (
          <Link to={`/challenges/${c.slug}`} key={c.id} className="challenge-card">
            <div className={`tag tag-${c.difficulty}`}>{c.difficulty}</div>
            <div className="category">{c.category}</div>
            <h3>{c.title}</h3>
            <p>{c.objective}</p>
            <div className="meta">⏱ {Math.round(c.time_limit_sec / 60)} min</div>
          </Link>
        ))}
      </div>
    </div>
  );
}
