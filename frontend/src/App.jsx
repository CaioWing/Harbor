import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "./context/AuthContext";
import LoginPage from "./pages/LoginPage";
import DashboardPage from "./pages/DashboardPage";
import DevicesPage from "./pages/DevicesPage";
import DeviceDetailsPage from "./pages/DeviceDetailsPage";
import DeploymentsPage from "./pages/DeploymentsPage";
import DeploymentDetailsPage from "./pages/DeploymentDetailsPage";
import AuditPage from "./pages/AuditPage";

function NotFound() {
  return <Navigate to="/" replace />;
}

export default function App() {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return (
    <Routes>
      <Route path="/" element={<DashboardPage />} />
      <Route path="/devices" element={<DevicesPage />} />
      <Route path="/devices/:id" element={<DeviceDetailsPage />} />
      <Route path="/deployments" element={<DeploymentsPage />} />
      <Route path="/deployments/:id" element={<DeploymentDetailsPage />} />
      <Route path="/audit" element={<AuditPage />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}
