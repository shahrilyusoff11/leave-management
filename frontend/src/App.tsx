import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import Login from './pages/Login';
import DashboardLayout from './layouts/DashboardLayout';
import Dashboard from './pages/Dashboard';
import RequestLeave from './pages/RequestLeave';
import MyLeaves from './pages/MyLeaves';
import TeamLeaves from './pages/TeamLeaves';
import UserManagement from './pages/UserManagement';
import AuditLogs from './pages/AuditLogs';
import Reports from './pages/Reports';
import HolidayManagement from './pages/HolidayManagement';
import Profile from './pages/Profile';
import HRLeaves from './pages/HRLeaves';
import SystemSettings from './pages/SystemSettings';
import LeaveTypeSettings from './pages/LeaveTypeSettings';

function App() {
  return (
    <Router>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />

          <Route element={<DashboardLayout />}>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/profile" element={<Profile />} />
            <Route path="/request-leave" element={<RequestLeave />} />
            <Route path="/my-leaves" element={<MyLeaves />} />
            <Route path="/team-leaves" element={<TeamLeaves />} />
            <Route path="/hr-leaves" element={<HRLeaves />} />
            <Route path="/users" element={<UserManagement />} />
            <Route path="/audit-logs" element={<AuditLogs />} />
            <Route path="/reports" element={<Reports />} />
            <Route path="/holidays" element={<HolidayManagement />} />
            <Route path="/settings" element={<SystemSettings />} />
            <Route path="/leave-type-settings" element={<LeaveTypeSettings />} />
          </Route>

          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </AuthProvider>
    </Router>
  );
}

export default App;

