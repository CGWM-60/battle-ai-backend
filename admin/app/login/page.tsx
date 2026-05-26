"use client";

import { useEffect, useState } from "react";

export default function LoginPage() {
  const [error, setError] = useState("");

  useEffect(() => {
    setError(new URLSearchParams(window.location.search).get("error") ?? "");
  }, []);

  return (
    <main className="login-body">
      <section className="login-panel">
        <h1>Admin</h1>
        <p>Interface de gestion backend.</p>
        {error ? <div className="alert error">{error}</div> : null}
        <form method="post" action="/admin/login">
          <label>
            Login
            <input name="username" autoComplete="username" required />
          </label>
          <label>
            Password
            <input name="password" type="password" autoComplete="current-password" required />
          </label>
          <button className="primary" type="submit">
            Se connecter
          </button>
        </form>
      </section>
    </main>
  );
}
