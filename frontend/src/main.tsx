import { createRoot } from "react-dom/client";
import "./index.css";

const apiOrigin = import.meta.env.VITE_API_ORIGIN ?? "http://localhost:8083";

createRoot(document.getElementById("root")!).render(
  <main className="p-8">
    <h1 className="text-2xl font-semibold">エスカレシミュレーター</h1>
    <p className="mt-2">API: {apiOrigin}</p>
  </main>,
);
