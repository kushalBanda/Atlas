import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { NavRail } from "./components/NavRail";
import { TraceList } from "./pages/TraceList";
import { TraceDetail } from "./pages/TraceDetail";
import { Discovery } from "./pages/Discovery";

export default function App() {
  return (
    <BrowserRouter>
      <div className="flex min-h-screen">
        <NavRail />
        <Routes>
          <Route path="/" element={<Navigate to="/traces" replace />} />
          <Route path="/traces" element={<TraceList />} />
          <Route path="/traces/:traceId" element={<TraceDetail />} />
          <Route path="/discovery" element={<Discovery />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}
