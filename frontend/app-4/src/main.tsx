import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

fetch('/config.json')
  .then((res) => res.json())
  .then((config) => {
    (window as any).__RUNTIME_CONFIG__ = config;
    createRoot(document.getElementById('root')!).render(
      <StrictMode>
        <App />
      </StrictMode>,
    )
  })
  .catch((err) => {
    console.error("Failed to load /config.json", err);
    document.body.innerHTML = "<div style='padding:20px;color:red;font-family:sans-serif'>Failed to load application configuration.</div>";
  });
