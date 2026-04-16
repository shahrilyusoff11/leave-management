import React from 'react';
import { NavLink } from 'react-router-dom';
import {
    LayoutDashboard,
    CalendarDays,
    PlusCircle,
    Users,
    ShieldCheck,
    X,
    FileText,
    Settings,
    User,
    Layers,
    Building
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { cn } from '../utils/cn';
import type { UserRole } from '../types';
import { canManageTeam } from '../utils/roles';

interface SidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

// Role hierarchy helpers
const isManager = (role?: UserRole) =>
    role === 'manager' || role === 'hod' || role === 'hr' || role === 'admin' || role === 'sysadmin';

const isHR = (role?: UserRole) =>
    role === 'hr' || role === 'admin' || role === 'sysadmin';

const isAdmin = (role?: UserRole) =>
    role === 'admin' || role === 'sysadmin';

const isSysAdmin = (role?: UserRole) =>
    role === 'sysadmin';



const Sidebar: React.FC<SidebarProps> = ({ isOpen, onClose }) => {
    const { user } = useAuth();

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



                    {/* Navigation */}
                    <nav className="flex-1 px-4 py-4 space-y-1 overflow-y-auto">
                        <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 px-2">Menu</div>

                        <NavItem to="/dashboard" icon={LayoutDashboard}>Dashboard</NavItem>
                        <NavItem to="/my-leaves" icon={CalendarDays}>My Leaves</NavItem>
                        {canManageTeam(user?.role) && (
                            <NavItem to="/delegations" icon={User}>My Delegations</NavItem>
                        )}
                        <NavItem to="/request-leave" icon={PlusCircle}>Request Leave</NavItem>

                        {/* Manager/HOD: Team management */}
                        {isManager(user?.role) && (
                            <>
                                <div className="mt-6 mb-2 px-2 text-xs font-semibold text-slate-400 uppercase tracking-wider">Management</div>
                                <NavItem to="/team-leaves" icon={Users}>Team Requests</NavItem>
                                <NavItem to="/team-calendar" icon={CalendarDays}>Team Calendar</NavItem>
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
                                <NavItem to="/admin/blackout-dates" icon={CalendarDays}>Blackout Dates</NavItem>
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

                </div>
            </aside>
        </>
    );
};

export default Sidebar;
