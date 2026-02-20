import React, { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';
import { useAuth } from '../context/AuthContext';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { useToast } from '../components/ui/Toast';

const requestSchema = z.object({
    leave_type: z.enum(['annual', 'sick', 'maternity', 'paternity', 'emergency', 'unpaid', 'unrecorded', 'hospitalization']),
    start_date: z.string().min(1, 'Start date is required'),
    end_date: z.string().min(1, 'End date is required'),
    is_half_day: z.boolean().optional(),
    half_day_period: z.enum(['AM', 'PM', '']).optional(),
    reason: z.string().min(1, 'Reason is required'),
    special_leave_type: z.string().optional(),
    attachment_url: z.string().optional(),
})
    .refine((data) => {
        if (data.leave_type === 'unrecorded' && !data.special_leave_type) {
            return false;
        }
        return true;
    }, {
        message: "Specific reason is required for Unrecorded Leave",
        path: ["special_leave_type"],
    })
    .refine((data) => {
        if (data.is_half_day && !data.half_day_period) {
            return false;
        }
        return true;
    }, {
        message: "Session (AM/PM) is required for Half-Day leave",
        path: ["half_day_period"],
    });

type RequestFormData = z.infer<typeof requestSchema>;

interface PublicHoliday {
    id: string;
    name: string;
    date: string;
}

interface LeaveTypeConfig {
    leave_type: string;
    is_active: boolean;
    requires_attachment: boolean;
}

import type { BlackoutDate } from '../types';

const RequestLeave: React.FC = () => {
    const navigate = useNavigate();
    const { showToast } = useToast();
    const { user } = useAuth();
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [uploading, setUploading] = useState(false);
    const [holidays, setHolidays] = useState<PublicHoliday[]>([]);
    const [leaveConfigs, setLeaveConfigs] = useState<LeaveTypeConfig[]>([]);
    const [blackoutDates, setBlackoutDates] = useState<BlackoutDate[]>([]);
    const { register, handleSubmit, setValue, formState: { errors }, watch } = useForm<RequestFormData>({
        resolver: zodResolver(requestSchema),
        defaultValues: {
            leave_type: 'annual'
        }
    });

    const startDate = watch('start_date');
    const endDate = watch('end_date');
    const leaveType = watch('leave_type');
    const isHalfDay = watch('is_half_day');

    // Sync end date to start date when half day is selected
    useEffect(() => {
        if (isHalfDay && startDate) {
            setValue('end_date', startDate);
        }
    }, [isHalfDay, startDate, setValue]);

    // Fetch public holidays and leave configs on mount
    useEffect(() => {
        const fetchData = async () => {
            try {
                const currentYear = new Date().getFullYear();
                const [holidaysRes, configsRes, blackoutRes] = await Promise.all([
                    api.get(`/public-holidays?year=${currentYear}`),
                    api.get('/leave-type-configs'),
                    api.get('/blackout-dates')
                ]);

                if (Array.isArray(holidaysRes.data)) {
                    setHolidays(holidaysRes.data);
                }

                if (Array.isArray(configsRes.data)) {
                    setLeaveConfigs(configsRes.data);
                }

                if (Array.isArray(blackoutRes.data)) {
                    setBlackoutDates(blackoutRes.data);
                }
            } catch (err) {
                console.error('Failed to fetch data', err);
            }
        };
        fetchData();
    }, []);

    // Check if a date is a public holiday
    const isPublicHoliday = (date: Date): boolean => {
        // Format as YYYY-MM-DD using local timezone
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const dateStr = `${year}-${month}-${day}`;

        return holidays.some(h => {
            // Extract just the date part from the holiday
            const holidayDate = h.date.split('T')[0];
            return holidayDate === dateStr;
        });
    };

    const calculateDuration = () => {
        if (!startDate || !endDate) return 0;
        if (isHalfDay) return 0.5;
        const start = new Date(startDate);
        const end = new Date(endDate);
        if (isNaN(start.getTime()) || isNaN(end.getTime())) return 0;
        if (end < start) return 0;

        // For maternity/paternity, count all calendar days
        if (leaveType === 'maternity' || leaveType === 'paternity') {
            const days = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;
            return days;
        }

        // Count working days (exclude weekends and public holidays)
        let workingDays = 0;
        const current = new Date(start);
        while (current <= end) {
            const dayOfWeek = current.getDay();
            // Skip weekends
            if (dayOfWeek !== 0 && dayOfWeek !== 6) {
                // Skip public holidays
                if (!isPublicHoliday(current)) {
                    workingDays++;
                }
            }
            current.setDate(current.getDate() + 1);
        }

        return Math.max(workingDays, 1); // At least 1 day for same-day leave
    };

    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        const formData = new FormData();
        formData.append('file', file);

        setUploading(true);
        try {
            const response = await api.post('/upload', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
            });
            setValue('attachment_url', response.data.url);
            showToast('File uploaded successfully', 'success');
        } catch (err) {
            showToast('Failed to upload file', 'error');
            console.error(err);
        } finally {
            setUploading(false);
        }
    };

    const checkBlackoutDates = (start: Date, end: Date, type: string) => {
        const overlapping = blackoutDates.filter(bd => {
            const bdStart = new Date(bd.start_date);
            const bdEnd = new Date(bd.end_date);
            // Ignore time for comparison, just compare dates
            bdStart.setHours(0, 0, 0, 0);
            bdEnd.setHours(23, 59, 59, 999);
            return end >= bdStart && start <= bdEnd;
        });

        for (const bd of overlapping) {
            if (bd.apply_to_all) {
                return `Leave period overlaps with a company-wide blackout date: ${bd.reason}`;
            }
            if (bd.department_id && user?.department_id === bd.department_id) {
                return `Leave period overlaps with a department blackout date: ${bd.reason}`;
            }
            if (bd.leave_type && bd.leave_type === type) {
                return `Leave period overlaps with a blackout date for this leave type: ${bd.reason}`;
            }
        }
        return null;
    };

    const onSubmit = async (data: RequestFormData) => {
        setIsLoading(true);
        setError(null);
        try {
            // Convert string dates to ISO strings for backend if needed, but backend takes Time?
            // Backend model takes time.Time. JSON unmarshaler usually handles RFC3339.
            // Native date input gives YYYY-MM-DD.
            // We need to append time or let backend handle it.
            // Validate attachment requirement based on config
            const selectedConfig = leaveConfigs.find(c => c.leave_type === data.leave_type);
            if (selectedConfig?.requires_attachment && !data.attachment_url) {
                setError(`Attachment is required for ${data.leave_type} leave`);
                setIsLoading(false);
                return;
            }

            const startD = new Date(data.start_date);
            const endD = new Date(data.end_date);
            startD.setHours(0, 0, 0, 0);
            endD.setHours(23, 59, 59, 999);

            const blackoutError = checkBlackoutDates(startD, endD, data.leave_type);
            if (blackoutError) {
                setError(blackoutError);
                setIsLoading(false);
                return;
            }

            const payload = {
                ...data,
                start_date: new Date(data.start_date).toISOString(),
                end_date: new Date(data.end_date).toISOString(),
                duration_days: calculateDuration(), // Backend might recalculate this
                special_leave_type: data.leave_type === 'unrecorded' ? data.special_leave_type : undefined,
            };

            await api.post('/leave-requests', payload);
            showToast('Leave request submitted successfully', 'success');
            navigate('/dashboard');
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to submit request');
        } finally {
            setIsLoading(false);
        }
    };


    return (
        <div className="max-w-2xl mx-auto">
            <h1 className="text-2xl font-bold text-slate-900 mb-6">Request Leave</h1>
            <Card>
                <CardHeader>
                    <CardTitle>New Leave Request</CardTitle>
                </CardHeader>
                <CardContent>
                    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                        {error && (
                            <div className="p-3 rounded-lg bg-red-50 text-red-600 text-sm border border-red-100">
                                {error}
                            </div>
                        )}

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">Leave Type</label>
                                <select
                                    {...register('leave_type')}
                                    className="flex h-10 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2"
                                >
                                    {leaveConfigs
                                        .filter(c => c.is_active)
                                        .map(c => (
                                            <option key={c.leave_type} value={c.leave_type}>
                                                {c.leave_type.charAt(0).toUpperCase() + c.leave_type.slice(1).replace('_', ' ')}
                                            </option>
                                        ))
                                    }
                                    {/* Fallback if configs not loaded yet */}
                                    {leaveConfigs.length === 0 && <option value="annual">Annual Leave</option>}
                                </select>
                                {errors.leave_type && <p className="mt-1 text-sm text-red-500">{errors.leave_type.message}</p>}
                            </div>

                            {leaveType === 'unrecorded' && (
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">Specific Reason</label>
                                    <Input
                                        {...register('special_leave_type')}
                                        placeholder="e.g. Marriage, Hajj, Compassionate"
                                        error={errors.special_leave_type?.message}
                                    />
                                    <p className="mt-1 text-xs text-slate-500">Please specify the type of unrecorded leave.</p>
                                </div>
                            )}

                            <div className="flex items-center gap-2 mb-2">
                                <input
                                    type="checkbox"
                                    id="is_half_day"
                                    className="h-4 w-4 rounded border-slate-300 text-brand-600 focus:ring-brand-500"
                                    {...register('is_half_day')}
                                />
                                <label htmlFor="is_half_day" className="text-sm font-medium text-slate-700">
                                    Apply for Half-Day
                                </label>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <Input
                                    type="date"
                                    label="Start Date"
                                    {...register('start_date')}
                                    min={new Date().toISOString().split('T')[0]}
                                    error={errors.start_date?.message}
                                />
                                {!isHalfDay ? (
                                    <Input
                                        type="date"
                                        label="End Date"
                                        {...register('end_date')}
                                        min={startDate || new Date().toISOString().split('T')[0]}
                                        error={errors.end_date?.message}
                                    />
                                ) : (
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 mb-1">Session</label>
                                        <select
                                            {...register('half_day_period')}
                                            className="flex h-10 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2"
                                        >
                                            <option value="">Select Session...</option>
                                            <option value="AM">Morning (AM)</option>
                                            <option value="PM">Afternoon (PM)</option>
                                        </select>
                                        {errors.half_day_period && <p className="mt-1 text-sm text-red-500">{errors.half_day_period.message}</p>}
                                    </div>
                                )}
                            </div>

                            {startDate && endDate && (
                                <div className="p-3 bg-blue-50 text-blue-700 text-sm rounded-lg flex items-center justify-between">
                                    <span>Estimated Duration:</span>
                                    <span className="font-bold">{calculateDuration()} days</span>
                                </div>
                            )}

                            <div>
                                <label className="block text-sm font-medium text-slate-700 mb-1">Reason</label>
                                <textarea
                                    {...register('reason')}
                                    className="flex min-h-[100px] w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2"
                                    placeholder="Please allow me to take leave because..."
                                />
                                {errors.reason && <p className="mt-1 text-sm text-red-500">{errors.reason.message}</p>}
                            </div>

                            {leaveType && (
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                        Attachment
                                        {leaveConfigs.find(c => c.leave_type === leaveType)?.requires_attachment ? ' (Required)' : ' (Optional)'}
                                    </label>
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="file"
                                            onChange={handleFileUpload}
                                            className="block w-full text-sm text-slate-500
                                                file:mr-4 file:py-2 file:px-4
                                                file:rounded-full file:border-0
                                                file:text-sm file:font-semibold
                                                file:bg-brand-50 file:text-brand-700
                                                hover:file:bg-brand-100"
                                            accept=".jpg,.jpeg,.png,.pdf,.doc,.docx"
                                        />
                                        {uploading && <span className="text-sm text-slate-500">Uploading...</span>}
                                    </div>
                                    <p className="mt-1 text-xs text-slate-500">Allowed: jpg, png, pdf, doc. Max 5MB.</p>
                                </div>
                            )}
                        </div>

                        <div className="flex gap-3 justify-end">
                            <Button type="button" variant="ghost" onClick={() => navigate('/dashboard')}>Cancel</Button>
                            <Button type="submit" isLoading={isLoading}>Submit Request</Button>
                        </div>
                    </form>
                </CardContent>
            </Card>
        </div>
    );
};

export default RequestLeave;
