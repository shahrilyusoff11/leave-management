import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Plus, Calendar, Trash2 } from 'lucide-react';
import api from '../services/api';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { ConfirmationModal } from '../components/ui/ConfirmationModal';
import { format } from 'date-fns';
import { useToast } from '../components/ui/Toast';
import type { BlackoutDate, Department, LeaveType } from '../types';

const BlackoutDateManagement: React.FC = () => {
    const { showToast } = useToast();
    const [blackoutDates, setBlackoutDates] = useState<BlackoutDate[]>([]);
    const [loading, setLoading] = useState(true);
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

    // Confirmation Modal State
    const [confirmModalOpen, setConfirmModalOpen] = useState(false);
    const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
    const [processingId, setProcessingId] = useState<string | null>(null);

    const fetchBlackoutDates = async () => {
        setLoading(true);
        try {
            const response = await api.get('/blackout-dates');
            if (Array.isArray(response.data)) {
                setBlackoutDates(response.data);
            }
        } catch (error) {
            console.error("Failed to fetch blackout dates", error);
            showToast('Failed to load blackout dates', 'error');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchBlackoutDates();
    }, []);

    const initiateDelete = (id: string) => {
        setPendingDeleteId(id);
        setConfirmModalOpen(true);
    };

    const handleConfirmDelete = async () => {
        if (!pendingDeleteId) return;

        setProcessingId(pendingDeleteId);
        try {
            await api.delete(`/admin/blackout-dates/${pendingDeleteId}`);
            showToast('Blackout date deleted', 'success');
            fetchBlackoutDates();
        } catch (error) {
            console.error("Failed to delete blackout date", error);
            showToast('Failed to delete blackout date', 'error');
        } finally {
            setProcessingId(null);
            setConfirmModalOpen(false);
            setPendingDeleteId(null);
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Blackout Dates</h1>
                    <p className="text-slate-500 mt-1">Manage periods where leave applications are restricted</p>
                </div>
                <Button onClick={() => setIsCreateModalOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Blackout Date
                </Button>
            </div>

            <Card>
                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="px-6 py-4 font-semibold text-slate-600">Date Range</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Reason</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Scope</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {loading ? (
                                <tr>
                                    <td colSpan={4} className="px-6 py-12 text-center text-slate-500">
                                        Loading blackout dates...
                                    </td>
                                </tr>
                            ) : blackoutDates.length === 0 ? (
                                <tr>
                                    <td colSpan={4} className="px-6 py-12 text-center text-slate-500">
                                        No blackout dates configured
                                    </td>
                                </tr>
                            ) : (
                                blackoutDates.map((bd) => (
                                    <tr key={bd.id} className="hover:bg-slate-50 transition-colors">
                                        <td className="px-6 py-4 font-medium text-slate-900">
                                            <div className="flex items-center gap-2">
                                                <Calendar className="h-4 w-4 text-slate-400" />
                                                {format(new Date(bd.start_date), 'MMM d, yyyy')} - {format(new Date(bd.end_date), 'MMM d, yyyy')}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 text-slate-600">
                                            {bd.reason}
                                        </td>
                                        <td className="px-6 py-4 text-slate-600">
                                            {bd.apply_to_all ? (
                                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-50 text-red-700">
                                                    All Staff
                                                </span>
                                            ) : bd.department_id ? (
                                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
                                                    Dept: {bd.department?.name || 'Unknown'}
                                                </span>
                                            ) : bd.leave_type ? (
                                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-50 text-purple-700">
                                                    Leave: {bd.leave_type.toUpperCase()}
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-50 text-gray-700">
                                                    Unknown Scope
                                                </span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                className="text-red-500 hover:bg-red-50 hover:text-red-600"
                                                onClick={() => initiateDelete(bd.id)}
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </Card>

            {isCreateModalOpen && (
                <CreateBlackoutModal
                    isOpen={isCreateModalOpen}
                    onClose={() => setIsCreateModalOpen(false)}
                    onSuccess={fetchBlackoutDates}
                    showToast={showToast}
                />
            )}

            <ConfirmationModal
                isOpen={confirmModalOpen}
                onClose={() => setConfirmModalOpen(false)}
                onConfirm={handleConfirmDelete}
                title="Delete Blackout Date"
                message="Are you sure you want to delete this blackout date? Staff will be able to apply for leave during this period again."
                confirmText="Delete Blackout Date"
                type="danger"
                isLoading={!!processingId}
            />
        </div>
    );
};

const CreateBlackoutModal = ({ isOpen, onClose, onSuccess, showToast }: { isOpen: boolean, onClose: () => void, onSuccess: () => void, showToast: (msg: string, type: 'success' | 'error') => void }) => {
    const { register, handleSubmit, watch, formState: { errors, isSubmitting } } = useForm({
        defaultValues: {
            scope: 'all', // 'all', 'department', 'leave_type'
            department_id: '',
            leave_type: '',
            start_date: '',
            end_date: '',
            reason: ''
        }
    });

    const [error, setError] = useState('');
    const [departments, setDepartments] = useState<Department[]>([]);
    const [leaveTypes, setLeaveTypes] = useState<any[]>([]);

    const scope = watch('scope');

    useEffect(() => {
        const fetchOptions = async () => {
            try {
                if (scope === 'department') {
                    const res = await api.get('/hr/departments');
                    setDepartments(res.data);
                } else if (scope === 'leave_type') {
                    const res = await api.get('/leave-type-configs');
                    setLeaveTypes(res.data);
                }
            } catch (err) {
                console.error("Failed to load options", err);
            }
        };
        fetchOptions();
    }, [scope]);

    const onSubmit = async (data: any) => {
        setError('');

        const payload = {
            start_date: new Date(data.start_date).toISOString(),
            end_date: new Date(data.end_date).toISOString(),
            reason: data.reason,
            apply_to_all: data.scope === 'all',
            department_id: data.scope === 'department' ? data.department_id : undefined,
            leave_type: data.scope === 'leave_type' ? data.leave_type : undefined,
        };

        if (payload.start_date > payload.end_date) {
            setError('Start date cannot be after end date');
            return;
        }

        try {
            await api.post('/admin/blackout-dates', payload);
            onSuccess();
            onClose();
            showToast('Blackout date added successfully', 'success');
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to create blackout date");
        }
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Add Blackout Date">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                {error && (
                    <div className="p-3 text-sm bg-red-50 text-red-600 rounded-lg">
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Start Date *</label>
                        <Input type="date" {...register('start_date', { required: 'Required' })} error={errors.start_date?.message as string} />
                    </div>
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">End Date *</label>
                        <Input type="date" {...register('end_date', { required: 'Required' })} error={errors.end_date?.message as string} />
                    </div>
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Reason *</label>
                    <Input {...register('reason', { required: 'Reason is required' })} error={errors.reason?.message as string} placeholder="e.g. Peak Season, Audit Week" />
                </div>

                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Scope</label>
                    <select
                        {...register('scope')}
                        className="w-full px-3 py-2 border rounded-md"
                    >
                        <option value="all">Apply to All Staff</option>
                        <option value="department">Apply by Department</option>
                        <option value="leave_type">Apply by Leave Type</option>
                    </select>
                </div>

                {scope === 'department' && (
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Select Department *</label>
                        <select
                            {...register('department_id', { required: scope === 'department' ? 'Required' : false })}
                            className="w-full px-3 py-2 border rounded-md"
                        >
                            <option value="">Select a department...</option>
                            {departments.map(d => (
                                <option key={d.id} value={d.id}>{d.name}</option>
                            ))}
                        </select>
                        {errors.department_id && <span className="text-xs text-red-500">{errors.department_id.message as string}</span>}
                    </div>
                )}

                {scope === 'leave_type' && (
                    <div className="space-y-1">
                        <label className="text-sm font-medium text-slate-700">Select Leave Type *</label>
                        <select
                            {...register('leave_type', { required: scope === 'leave_type' ? 'Required' : false })}
                            className="w-full px-3 py-2 border rounded-md"
                        >
                            <option value="">Select a leave type...</option>
                            {leaveTypes.map(lt => (
                                <option key={lt.id} value={lt.leave_type}>{lt.leave_type.toUpperCase()}</option>
                            ))}
                        </select>
                        {errors.leave_type && <span className="text-xs text-red-500">{errors.leave_type.message as string}</span>}
                    </div>
                )}

                <div className="flex justify-end gap-3 mt-6">
                    <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
                    <Button type="submit" isLoading={isSubmitting}>Save Blackout Date</Button>
                </div>
            </form>
        </Modal>
    );
};

export default BlackoutDateManagement;
