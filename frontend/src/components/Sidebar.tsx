import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import {
    LayoutDashboard,
    CalendarDays,
    PlusCircle,
    Users,
    ShieldCheck,
    LogOut,
    X,
    FileText,
    Settings,
    User,
    Layers,
    Building
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { cn } from '../utils/cn';
import { Button } from './ui/Button';
import type { UserRole } from '../types';

interface SidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

// Role hierarchy helpers
const isManager = (role?: UserRole) =>
    ['manager', 'hod', 'hr', 'admin', 'sysadmin'].includes(role || '');

const isHR = (role?: UserRole) =>
    role === 'hr' || role === 'admin' || role === 'sysadmin';

const isAdmin = (role?: UserRole) =>
    role === 'admin' || role === 'sysadmin';

const isSysAdmin = (role?: UserRole) =>
    role === 'sysadmin';

const getRoleLabel = (role?: UserRole) => {
    switch (role) {
        case 'sysadmin': return 'System Administrator';
        case 'admin': return 'Administrator';
        case 'hr': return 'Human Resources';
        case 'hod': return 'Head of Department';
        case 'manager': return 'Manager';
        case 'staff': return 'Staff';
        default: return role || 'Unknown';
    }
};

const Sidebar: React.FC<SidebarProps> = ({ isOpen, onClose }) => {
    const { user, logout } = useAuth();
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    const NavItem = ({ to, icon: Icon, children }: { to: string, icon: any, children: React.ReactNode }) => (
        <NavLink
            to={to}
            onClick={onClose}
            className={({ isActive }) => cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 group text-sm font-medium",
                isActive
                    ? "bg-brand-50 text-brand-600 shadow-sm"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
            )}
        >
            <Icon className="h-5 w-5" />
            {children}
        </NavLink>
    );

    return (
        <>
            {/* Mobile overlay */}
            {isOpen && (
                <div
                    className="fixed inset-0 bg-slate-900/20 backdrop-blur-sm z-40 lg:hidden"
                    onClick={onClose}
                />
            )}

            {/* Sidebar */}
            <aside className={cn(
                "fixed top-0 left-0 z-50 h-full w-64 bg-white border-r border-slate-200 shadow-xl lg:shadow-none transition-transform duration-300 lg:translate-x-0 lg:static",
                isOpen ? "translate-x-0" : "-translate-x-full"
            )}>
                <div className="flex flex-col h-full">
                    {/* Header */}
                    <div className="h-16 flex items-center px-6 border-b border-slate-100">
                        <div className="flex items-center gap-2">
                            <div className="h-8 w-8 rounded-lg bg-brand-600 flex items-center justify-center text-white font-bold">
                                LM
                            </div>
                            <span className="text-lg font-bold text-slate-900">LeaveSys</span>
                        </div>
                        <button className="ml-auto lg:hidden" onClick={onClose}>
                            <X className="h-5 w-5 text-slate-500" />
                        </button>
                    </div>

                    {/* User Info */}
                    <div className="p-4 mx-4 mt-4 rounded-xl bg-slate-50 border border-slate-100">
                        <p className="font-semibold text-sm text-slate-900">{user?.first_name} {user?.last_name}</p>
                        <p className="text-xs text-slate-500">{getRoleLabel(user?.role)}</p>
                        {(user as any)?.department_ref?.name && (
                            <p className="text-xs text-slate-400 mt-1 flex items-center gap-1">
                                <Building className="h-3 w-3" />
                                {(user as any).department_ref.name}
                            </p>
                        )}
                    </div>

                    {/* Navigation */}
                    <nav className="flex-1 px-4 py-4 space-y-1 overflow-y-auto">
                        <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 px-2">Menu</div>

                        <NavItem to="/dashboard" icon={LayoutDashboard}>Dashboard</NavItem>
                        <NavItem to="/profile" icon={User}>My Profile</NavItem>
                        <NavItem to="/my-leaves" icon={CalendarDays}>My Leaves</NavItem>
                        <NavItem to="/delegations" icon={User}>My Delegations</NavItem>
                        <NavItem to="/request-leave" icon={PlusCircle}>Request Leave</NavItem>

                        {/* Manager/HOD: Team management */}
                        {isManager(user?.role) && (
                            <>
                                <div className="mt-6 mb-2 px-2 text-xs font-semibold text-slate-400 uppercase tracking-wider">Management</div>
                                <NavItem to="/team-leaves" icon={Users}>Team Requests</NavItem>
                            </>
                        )}

                        {/* HR: User management and reports */}
                        {isHR(user?.role) && (
                            <>
                                <div className="mt-6 mb-2 px-2 text-xs font-semibold text-slate-400 uppercase tracking-wider">HR Administration</div>
                                <NavItem to="/users" icon={Users}>User Management</NavItem>
                                <NavItem to="/hr/departments" icon={Building}>Departments</NavItem>
                                <NavItem to="/hr-leaves" icon={CalendarDays}>All Leave Requests</NavItem>
                                <NavItem to="/reports" icon={FileText}>Reports</NavItem>
                            </>
                        )}

                        {/* Admin: System configuration */}
                        {isAdmin(user?.role) && (
                            <>
                                <div className="mt-6 mb-2 px-2 text-xs font-semibold text-slate-400 uppercase tracking-wider">System</div>
                                <NavItem to="/holidays" icon={CalendarDays}>Public Holidays</NavItem>
                                <NavItem to="/leave-type-settings" icon={Layers}>Leave Types</NavItem>
                                <NavItem to="/settings" icon={Settings}>System Settings</NavItem>
                            </>
                        )}

                        {/* SysAdmin: Audit and advanced */}
                        {isSysAdmin(user?.role) && (
                            <>
                                <NavItem to="/audit-logs" icon={ShieldCheck}>Audit Logs</NavItem>
                            </>
                        )}
                    </nav>

                    {/* Footer */}
                    <div className="p-4 border-t border-slate-100">
                        <Button variant="ghost" className="w-full justify-start text-red-600 hover:bg-red-50 hover:text-red-700" onClick={handleLogout}>
                            <LogOut className="mr-2 h-4 w-4" />
                            Sign Out
                        </Button>
                    </div>
                </div>
            </aside>
        </>
    );
};

export default Sidebar;
