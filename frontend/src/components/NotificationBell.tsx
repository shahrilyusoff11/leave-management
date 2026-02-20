import React, { useState, useRef, useEffect } from 'react';
import { Bell, Check, Circle, Activity, FileText, Info } from 'lucide-react';
import { useNotifications } from '../context/NotificationContext';
import type { Notification } from '../context/NotificationContext';
import { formatDistanceToNow } from 'date-fns';

const NotificationBell: React.FC = () => {
    const { notifications, unreadCount, markAsRead, markAllAsRead } = useNotifications();
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

    const getIcon = (type: string) => {
        switch (type) {
            case 'leave': return <FileText className="h-4 w-4 text-brand-600" />;
            case 'workflow': return <Activity className="h-4 w-4 text-purple-600" />;
            default: return <Info className="h-4 w-4 text-slate-600" />;
        }
    };

    const handleNotificationClick = (notif: Notification) => {
        if (!notif.is_read) {
            markAsRead(notif.id);
        }
        setIsOpen(false);
        // We could also navigate to the specific entity here if needed
    };

    return (
        <div className="relative" ref={dropdownRef}>
            <button
                type="button"
                className="relative p-2 text-slate-500 hover:bg-slate-100 focus:bg-slate-100 rounded-full transition-colors"
                onClick={() => setIsOpen(!isOpen)}
                aria-label="Notifications"
            >
                <Bell className="h-5 w-5" />
                {unreadCount > 0 && (
                    <span className="absolute top-1 right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] font-bold text-white ring-2 ring-white">
                        {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                )}
            </button>

            {isOpen && (
                <div className="absolute right-0 mt-2 w-80 md:w-96 bg-white rounded-xl shadow-xl border border-slate-200 z-50 overflow-hidden transform opacity-100 scale-100 transition-all origin-top-right">
                    <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50/80 backdrop-blur-sm">
                        <h3 className="font-semibold text-slate-900">Notifications</h3>
                        {unreadCount > 0 && (
                            <button
                                onClick={() => markAllAsRead()}
                                className="text-xs font-medium text-brand-600 hover:text-brand-700 flex items-center gap-1"
                            >
                                <Check className="h-3 w-3" />
                                Mark all as read
                            </button>
                        )}
                    </div>

                    <div className="max-h-[400px] overflow-y-auto w-full">
                        {notifications.length === 0 ? (
                            <div className="py-8 text-center flex flex-col items-center justify-center">
                                <div className="h-12 w-12 rounded-full bg-slate-100 flex items-center justify-center mb-3">
                                    <Bell className="h-6 w-6 text-slate-400" />
                                </div>
                                <p className="text-slate-500 font-medium">No notifications</p>
                                <p className="text-sm text-slate-400">You're all caught up!</p>
                            </div>
                        ) : (
                            <ul className="divide-y divide-slate-100">
                                {notifications.map((notif) => (
                                    <li
                                        key={notif.id}
                                        className={`p-4 hover:bg-slate-50 transition-colors cursor-pointer flex gap-3 ${!notif.is_read ? 'bg-brand-50/50' : ''}`}
                                        onClick={() => handleNotificationClick(notif)}
                                    >
                                        <div className={`mt-0.5 rounded-full p-1.5 flex-shrink-0 bg-white shadow-sm border ${notif.is_read ? 'border-slate-100' : 'border-brand-200'}`}>
                                            {getIcon(notif.type)}
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-start justify-between gap-1">
                                                <p className={`text-sm font-medium ${!notif.is_read ? 'text-slate-900' : 'text-slate-700'} truncate`}>
                                                    {notif.title}
                                                </p>
                                                {!notif.is_read && (
                                                    <Circle className="h-2 w-2 fill-brand-600 text-brand-600 mt-1.5 flex-shrink-0" />
                                                )}
                                            </div>
                                            <p className={`text-sm mt-0.5 break-words line-clamp-2 ${!notif.is_read ? 'text-slate-700' : 'text-slate-500'}`}>
                                                {notif.message}
                                            </p>
                                            <p className="text-xs text-slate-400 mt-2 font-medium">
                                                {formatDistanceToNow(new Date(notif.created_at), { addSuffix: true })}
                                            </p>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};

export default NotificationBell;
