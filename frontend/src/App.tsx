import { BrowserRouter, Route, Routes, useLocation } from "react-router-dom";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { NavRail } from "./components/NavRail";
import { Home } from "./pages/Home";
import { TraceList } from "./pages/TraceList";
import { TraceDetail } from "./pages/TraceDetail";
import { Discovery } from "./pages/Discovery";
import { RunList } from "./pages/RunList";
import { RunDetail } from "./pages/RunDetail";
import { SessionList } from "./pages/SessionList";

// Keyed by pathname so navigating away from a crashed page remounts a
// fresh ErrorBoundary instead of carrying the crash state into the next
// route — a boundary's own state only resets when React actually
// discards and recreates the instance.
function AppRoutes() {
  const location = useLocation();
  return (
    <ErrorBoundary key={location.pathname}>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/traces" element={<TraceList />} />
        <Route path="/traces/:traceId" element={<TraceDetail />} />
        <Route path="/discovery" element={<Discovery />} />
        <Route path="/runs" element={<RunList />} />
        <Route path="/runs/:runId" element={<RunDetail />} />
        <Route path="/sessions" element={<SessionList />} />
      </Routes>
    </ErrorBoundary>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <div className="flex min-h-screen">
        <NavRail />
        <AppRoutes />
      </div>
    </BrowserRouter>
  );
}
