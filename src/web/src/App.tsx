import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

function App() {
  return (
    <div>
      <h1>Werd</h1>
      <p>Dashboard coming soon.</p>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
