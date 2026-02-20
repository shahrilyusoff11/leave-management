import React, { useState } from 'react';
import { Outlet, Navigate } from 'react-router-dom';
import { Menu } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import Sidebar from '../components/Sidebar';
import NotificationBell from '../components/NotificationBell';
import UserMenu from '../components/UserMenu';

const DashboardLayout: React.FC = () => {
    const { isAuthenticated, isLoading } = useAuth();
    const [isSidebarOpen, setSidebarOpen] = useState(false);

    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-50">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
        );
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    return (
        <div className="flex h-screen overflow-hidden bg-slate-50">
            <Sidebar isOpen={isSidebarOpen} onClose={() => setSidebarOpen(false)} />

            <div className="flex-1 flex flex-col min-w-0 transition-all duration-300">
                {/* Header (Mobile & Desktop) */}
                <header className="h-16 bg-white border-b border-slate-200 flex items-center px-4 md:px-8 sticky top-0 z-30">
                    <button
                        type="button"
                        className="p-2 -ml-2 rounded-lg text-slate-500 hover:bg-slate-100 lg:hidden"
                        onClick={() => setSidebarOpen(true)}
                    >
                        <Menu className="h-6 w-6" />
                    </button>
                    <span className="ml-3 font-semibold text-slate-900 lg:hidden">Leave Management</span>
                    <div className="ml-auto flex items-center gap-2">
                        <NotificationBell />
                        <div className="w-px h-6 bg-slate-200 mx-1 hidden md:block" />
                        <UserMenu />
                    </div>
                </header>

                {/* Main Content */}
                <main className="flex-1 p-4 lg:p-8 overflow-y-auto">
                    <div className="max-w-7xl mx-auto">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    );
};

export default DashboardLayout;
