import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Save, AlertTriangle, Building, Bell, Clock, Calendar, CheckCircle } from 'lucide-react';
import api from '../services/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';

interface SystemConfig {
    company_name: string;
    timezone: string;
    date_format: string;
    is_email_enabled: boolean;
    max_carry_forward_days: number; // Deprecated
    working_days: string[];
    escalation_days: number; // Deprecated
}

const SystemSettings: React.FC = () => {
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [activeTab, setActiveTab] = useState<'general' | 'notifications'>('general');

    const { register, handleSubmit, reset, watch, setValue, formState: { errors } } = useForm<SystemConfig>();
    const workingDays = watch('working_days') || [];

    const fetchConfig = async () => {
        setLoading(true);
        try {
            const response = await api.get('/admin/config');
            reset(response.data);
        } catch (err) {
            console.error("Failed to fetch config", err);
            setError("Failed to load system configuration");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchConfig();
    }, []);

    const onSubmit = async (data: SystemConfig) => {
        setSaving(true);
        setMessage('');
        setError('');
        try {
            await api.put('/admin/config', data);
            setMessage('System configuration updated successfully');
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update configuration");
        } finally {
            setSaving(false);
        }
    };

    const toggleWorkingDay = (day: string) => {
        const currentDays = workingDays;
        if (currentDays.includes(day)) {
            setValue('working_days', currentDays.filter(d => d !== day));
        } else {
            setValue('working_days', [...currentDays, day]);
        }
    };

    const daysOfWeek = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">System Settings</h1>
                    <p className="text-slate-500 mt-1">Configure system-wide settings and preferences</p>
                </div>
                <Button
                    onClick={handleSubmit(onSubmit)}
                    isLoading={saving}
                    className="flex items-center gap-2"
                >
                    <Save className="h-4 w-4" />
                    Save Changes
                </Button>
            </div>

            {message && (
                <div className="p-4 bg-green-50 text-green-700 rounded-lg border border-green-200 flex items-center gap-2">
                    <CheckCircle className="h-4 w-4" />
                    {message}
                </div>
            )}
            {error && (
                <div className="p-4 bg-red-50 text-red-700 rounded-lg border border-red-200 flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4" />
                    {error}
                </div>
            )}

            <div className="flex space-x-1 bg-slate-100 p-1 rounded-lg w-full sm:w-auto overflow-x-auto">
                <button
                    onClick={() => setActiveTab('general')}
                    className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${activeTab === 'general' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600 hover:text-slate-900'
                        }`}
                >
                    <Building className="h-4 w-4" />
                    General
                </button>
                <button
                    onClick={() => setActiveTab('notifications')}
                    className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${activeTab === 'notifications' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600 hover:text-slate-900'
                        }`}
                >
                    <Bell className="h-4 w-4" />
                    Notifications
                </button>
            </div>

            <div className="grid grid-cols-1 gap-6">

                {/* General Settings */}
                {activeTab === 'general' && (
                    <Card>
                        <CardHeader>
                            <CardTitle>General Configuration</CardTitle>
                            <CardDescription>Basic system information and operational settings.</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Company Name</label>
                                    <Input
                                        {...register('company_name', { required: 'Required' })}
                                        placeholder="Enter company name"
                                    />
                                    {errors.company_name && <p className="text-xs text-red-500">{errors.company_name.message}</p>}
                                </div>
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Timezone</label>
                                    <div className="relative">
                                        <Clock className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                                        <select
                                            {...register('timezone')}
                                            className="w-full pl-9 pr-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                        >
                                            <option value="UTC">UTC</option>
                                            <option value="Asia/Kuala_Lumpur">Asia/Kuala_Lumpur (+08:00)</option>
                                            <option value="America/New_York">America/New_York (-05:00)</option>
                                            <option value="Europe/London">Europe/London (+00:00)</option>
                                        </select>
                                    </div>
                                </div>
                                <div className="space-y-1">
                                    <label className="text-sm font-medium text-slate-700">Date Format</label>
                                    <div className="relative">
                                        <Calendar className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                                        <select
                                            {...register('date_format')}
                                            className="w-full pl-9 pr-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                                        >
                                            <option value="YYYY-MM-DD">YYYY-MM-DD (2024-12-31)</option>
                                            <option value="DD/MM/YYYY">DD/MM/YYYY (31/12/2024)</option>
                                            <option value="MM/DD/YYYY">MM/DD/YYYY (12/31/2024)</option>
                                        </select>
                                    </div>
                                </div>
                            </div>

                            <div className="space-y-2 pt-4 border-t border-slate-200">
                                <label className="text-sm font-medium text-slate-700">Working Days</label>
                                <div className="flex flex-wrap gap-2">
                                    {daysOfWeek.map(day => (
                                        <button
                                            key={day}
                                            type="button"
                                            onClick={() => toggleWorkingDay(day)}
                                            className={`px-3 py-1.5 text-sm rounded-full border transition-colors ${workingDays.includes(day)
                                                    ? 'bg-brand-50 border-brand-200 text-brand-700 font-medium'
                                                    : 'bg-white border-slate-200 text-slate-500 hover:border-slate-300'
                                                }`}
                                        >
                                            {day}
                                        </button>
                                    ))}
                                </div>
                                <p className="text-xs text-slate-500">
                                    Selected days count towards leave duration calculation.
                                </p>
                            </div>
                        </CardContent>
                    </Card>
                )}

                {/* Notifications */}
                {activeTab === 'notifications' && (
                    <Card>
                        <CardHeader>
                            <CardTitle>Notification Settings</CardTitle>
                            <CardDescription>Manage how the system communicates with users.</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="flex items-center justify-between p-4 border border-slate-200 rounded-lg">
                                <div className="flex items-start gap-3">
                                    <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
                                        <Bell className="h-5 w-5" />
                                    </div>
                                    <div>
                                        <h4 className="font-medium text-slate-900">Email Notifications</h4>
                                        <p className="text-sm text-slate-500">
                                            Send emails for leave requests, approvals, and rejections.
                                        </p>
                                    </div>
                                </div>
                                <label className="relative inline-flex items-center cursor-pointer">
                                    <input
                                        type="checkbox"
                                        className="sr-only peer"
                                        {...register('is_email_enabled')}
                                    />
                                    <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-brand-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-brand-600"></div>
                                </label>
                            </div>
                        </CardContent>
                    </Card>
                )}
            </div>
        </div>
    );
};

export default SystemSettings;
