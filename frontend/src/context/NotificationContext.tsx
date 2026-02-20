import React, { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import api from '../services/api'; // Axios instance pointing to /api/v1
import { useAuth } from './AuthContext';

export interface Notification {
    id: string;
    title: string;
    message: string;
    type: 'leave' | 'workflow' | 'system';
    is_read: boolean;
    related_entity_id?: string;
    created_at: string;
}

interface NotificationContextType {
    notifications: Notification[];
    unreadCount: number;
    markAsRead: (id: string) => Promise<void>;
    markAllAsRead: () => Promise<void>;
}

const NotificationContext = createContext<NotificationContextType | undefined>(undefined);

export const useNotifications = () => {
    const context = useContext(NotificationContext);
    if (!context) {
        throw new Error('useNotifications must be used within a NotificationProvider');
    }
    return context;
};

export const NotificationProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const { isAuthenticated, token } = useAuth();
    const [notifications, setNotifications] = useState<Notification[]>([]);
    const [unreadCount, setUnreadCount] = useState<number>(0);

    // Initial fetch of notifications
    useEffect(() => {
        if (!isAuthenticated) return;

        const fetchInitialData = async () => {
            try {
                const [notifsRes, countRes] = await Promise.all([
                    api.get('/notifications?limit=20'),
                    api.get('/notifications/unread-count')
                ]);
                setNotifications(notifsRes.data);
                setUnreadCount(countRes.data.count);
            } catch (error) {
                console.error("Failed to fetch initial notifications", error);
            }
        };

        fetchInitialData();
    }, [isAuthenticated]);

    // Setup Server-Sent Events (SSE) connection
    useEffect(() => {
        if (!isAuthenticated || !token) return;

        // Ensure EventSource is created correctly based on the environment path
        const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

        // Pass the auth token as a URL param or establish a session 
        // Note: EventSource doesn't natively support custom headers like Authorization Bearer.
        // We'll append JWT to query string or rely on cookies. Since we use Bearer tokens:
        // Wait, standard SSE doesn't do Bearer. The safest polyfill or approach is via a custom fetch stream
        // To keep it clean, let's use the native EventSource but we have to handle auth.
        // If the backend `authMiddleware` expects headers only, this will fail.
        // Let's modify our logic to use standard SSE, and if backend blocks it, we might need a workaround.
        // For now, let's try standard fetch streams or just polling if SSE is blocked by JWT auth.
        // Actually, fetching with ReadableStream natively solves the JWT header issue:

        let shouldContinue = true;

        const connectSSE = async () => {
            try {
                const response = await fetch(`${apiUrl}/notifications/stream`, {
                    headers: {
                        'Authorization': `Bearer ${token}`,
                        'Accept': 'text/event-stream'
                    }
                });

                if (!response.body) return;
                const reader = response.body.getReader();
                const decoder = new TextDecoder('utf-8');

                while (shouldContinue) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value, { stream: true });
                    const lines = chunk.split('\n');

                    for (let i = 0; i < lines.length; i++) {
                        const line = lines[i];
                        if (line.startsWith('data: ')) {
                            const data = line.slice(6);
                            if (data === 'connected') continue;

                            try {
                                const newNotification: Notification = JSON.parse(data);
                                setNotifications(prev => [newNotification, ...prev]);
                                setUnreadCount(prev => prev + 1);

                                // Optional: You could trigger browser notification API here
                                if (Notification.permission === 'granted') {
                                    new window.Notification(newNotification.title, {
                                        body: newNotification.message,
                                        icon: '/vite.svg'
                                    });
                                }
                            } catch (e) {
                                console.error("Error parsing SSE data", e);
                            }
                        }
                    }
                }
            } catch (error) {
                console.error("SSE stream disconnected", error);
            }
        };

        connectSSE();

        // Request browser notification permission
        if ('Notification' in window && Notification.permission === 'default') {
            Notification.requestPermission();
        }

        return () => {
            shouldContinue = false;
        };
    }, [isAuthenticated, token]);

    const markAsRead = async (id: string) => {
        if (!notifications.find(n => n.id === id && !n.is_read)) return;

        try {
            await api.put(`/notifications/${id}/read`);
            setNotifications(prev => prev.map(n => n.id === id ? { ...n, is_read: true } : n));
            setUnreadCount(prev => Math.max(0, prev - 1));
        } catch (error) {
            console.error("Failed to mark as read", error);
        }
    };

    const markAllAsRead = async () => {
        if (unreadCount === 0) return;

        try {
            await api.put('/notifications/read-all');
            setNotifications(prev => prev.map(n => ({ ...n, is_read: true })));
            setUnreadCount(0);
        } catch (error) {
            console.error("Failed to mark all as read", error);
        }
    };

    return (
        <NotificationContext.Provider value={{ notifications, unreadCount, markAsRead, markAllAsRead }}>
            {children}
        </NotificationContext.Provider>
    );
};
