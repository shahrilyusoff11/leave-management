import axios from 'axios';

export interface LeaveTypeConfig {
    id: string;
    leave_type: string;
    base_entitlement: number;
    years_of_service_tiers?: Record<string, number>;
    prorate_type: 'none' | 'first_year' | 'continuous';
    allow_carry_forward: boolean;
    max_carry_forward_days: number;
    requires_attachment: boolean;
    min_advance_days: number;
    is_active: boolean;
    display_order: number;
    allow_negative_balance: boolean;
}

const api = axios.create({
    baseURL: '/api/v1',
    headers: {
        'Content-Type': 'application/json',
    },
});

api.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);

api.interceptors.response.use(
    (response) => response,
    (error) => {
        // Don't redirect if it's a login failure (401 on /login)
        if (error.response?.status === 401 && !error.config.url?.includes('/login')) {
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

export default api;
