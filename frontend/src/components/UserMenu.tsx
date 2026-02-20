import React, { useState, useRef, useEffect } from 'react';
import { User, LogOut, ChevronDown, Building, ShieldCheck } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';
import type { UserRole } from '../types';

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

const UserMenu: React.FC = () => {
    const { user, logout } = useAuth();
    const navigate = useNavigate();
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    // Click outside to close
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                setIsOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    const handleProfileClick = () => {
        setIsOpen(false);
        navigate('/profile');
    };

    if (!user) return null;

    // Get initials for avatar
    const initials = `${user.first_name?.[0] || ''}${user.last_name?.[0] || ''}`.toUpperCase();

    return (
        <div className="relative" ref={dropdownRef}>
            <button
                type="button"
                className="flex items-center gap-2 p-1.5 pl-2 rounded-full hover:bg-slate-100 transition-colors border border-transparent hover:border-slate-200"
                onClick={() => setIsOpen(!isOpen)}
                aria-expanded={isOpen}
            >
                <div className="h-8 w-8 rounded-full bg-brand-100 text-brand-700 font-bold text-xs flex items-center justify-center shadow-sm border border-brand-200">
                    {initials || <User className="h-4 w-4" />}
                </div>
                <div className="hidden md:flex flex-col items-start mr-1">
                    <span className="text-sm font-medium text-slate-700 leading-tight">
                        {user.first_name}
                    </span>
                    <span className="text-[10px] text-slate-500 leading-tight">
                        {getRoleLabel(user.role)}
                    </span>
                </div>
                <ChevronDown className="h-4 w-4 text-slate-400 hidden md:block" />
            </button>

            {isOpen && (
                <div className="absolute right-0 mt-2 w-64 bg-white rounded-xl shadow-xl border border-slate-200 z-50 overflow-hidden transform opacity-100 scale-100 transition-all origin-top-right">
                    {/* User Info Header */}
                    <div className="p-4 border-b border-slate-100 bg-slate-50/50">
                        <p className="font-semibold text-sm text-slate-900 truncate">
                            {user.first_name} {user.last_name}
                        </p>
                        <p className="text-xs text-slate-500 mt-0.5 truncate">{user.email}</p>

                        <div className="mt-3 flex flex-col gap-1.5">
                            <div className="flex items-center gap-1.5 text-xs text-brand-700 bg-brand-50 px-2 py-1 rounded w-fit border border-brand-100">
                                <ShieldCheck className="h-3 w-3" />
                                {getRoleLabel(user.role)}
                            </div>

                            {(user as any)?.department_ref?.name && (
                                <div className="flex items-center gap-1.5 text-xs text-slate-600 bg-slate-100 px-2 py-1 rounded w-fit border border-slate-200">
                                    <Building className="h-3 w-3" />
                                    <span className="truncate max-w-[180px]">{(user as any).department_ref.name}</span>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Actions */}
                    <div className="p-2">
                        <button
                            onClick={handleProfileClick}
                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 hover:text-brand-600 rounded-lg transition-colors"
                        >
                            <User className="h-4 w-4 text-slate-400" />
                            My Profile
                        </button>
                    </div>

                    <div className="p-2 border-t border-slate-100">
                        <button
                            onClick={handleLogout}
                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 rounded-lg transition-colors font-medium"
                        >
                            <LogOut className="h-4 w-4 text-red-500" />
                            Sign Out
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
};

export default UserMenu;
