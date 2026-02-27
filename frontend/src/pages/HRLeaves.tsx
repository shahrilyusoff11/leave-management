import React, { useEffect, useState } from 'react';
import { format } from 'date-fns';
import { Filter, GitBranch, FileText } from 'lucide-react';
import api from '../services/api';
import type { LeaveRequest } from '../types';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { getDisplayDuration, formatDuration } from '../utils/duration';
import WorkflowStateDisplay from '../components/WorkflowStateDisplay';
import { useAuth } from '../context/AuthContext';

const HRLeaves: React.FC = () => {
    const { user } = useAuth();
    const [requests, setRequests] = useState<LeaveRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [statusFilter, setStatusFilter] = useState('all');
    const [deptFilter, setDeptFilter] = useState('');
    const [yearFilter, setYearFilter] = useState(new Date().getFullYear().toString());
    const [showWorkflow, setShowWorkflow] = useState<Record<string, boolean>>({});

    const fetchRequests = async () => {
        setLoading(true);
        try {
            // Build query params
            const params = new URLSearchParams();
            if (statusFilter !== 'all') params.append('status', statusFilter);
            if (deptFilter) params.append('department', deptFilter);
            if (yearFilter) params.append('year', yearFilter);

            const response = await api.get(`/hr/leave-requests?${params.toString()}`);
            if (Array.isArray(response.data)) {
                setRequests(response.data);
            }
        } catch (error) {
            console.error("Failed to fetch HR leave requests", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchRequests();
    }, [statusFilter, yearFilter]); // Trigger on simple filters directly

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        fetchRequests();
    };

    const getStatusVariant = (status: string) => {
        switch (status) {
            case 'approved': return 'success';
            case 'rejected': return 'danger';
            case 'pending': return 'warning';
            case 'cancelled': return 'secondary';
            default: return 'default';
        }
    };

    // Helper to check if current user can approve/reject
    const canActionRequest = (req: LeaveRequest) => {
        if (req.status !== 'pending') return false;

        // If workflow state is present, enforce strict role check
        if (req.workflow_state?.current_step) {
            const requiredRole = req.workflow_state.current_step.responsible_role;
            // SysAdmin can always override (matching backend logic)
            if (user?.role === 'sysadmin') return true;
            return user?.role === requiredRole;
        }

        // Fallback: If no workflow state, allow SysAdmin/HR? (HR usually can act on HRLeaves page, but strict workflow says no)
        // Let's stick to strict compliance for consistency, or maybe allow HR if role matches?
        // HRLeaves page is for HR users.
        // If workflow step is "HR Verification", responsible_role='hr'. So user.role === 'hr' works.
        // If workflow step is "Manager Approval", HR CANNOT approve (unless SysAdmin).
        // So this logic works perfectly.
        return user?.role === 'sysadmin';
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">All Leave Requests</h1>
                    <p className="text-slate-500 mt-1">Global view of all employee leave requests</p>
                </div>
            </div>

            <Card className="p-4">
                <form onSubmit={handleSearch} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end mb-4">
                    <div>
                        <label className="block text-xs font-medium text-slate-700 mb-1">Status</label>
                        <div className="relative">
                            <select
                                className="w-full appearance-none pl-9 pr-8 h-10 border border-slate-300 rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all"
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value)}
                            >
                                <option value="all">All Status</option>
                                <option value="pending">Pending</option>
                                <option value="approved">Approved</option>
                                <option value="rejected">Rejected</option>
                                <option value="cancelled">Cancelled</option>
                            </select>
                            <Filter className="absolute left-3 top-3 h-4 w-4 text-slate-400" />
                        </div>
                    </div>

                    <div>
                        <label className="block text-xs font-medium text-slate-700 mb-1">Department</label>
                        <Input
                            placeholder="e.g. Engineering"
                            value={deptFilter}
                            onChange={(e) => setDeptFilter(e.target.value)}
                        />
                    </div>

                    <div>
                        <label className="block text-xs font-medium text-slate-700 mb-1">Year</label>
                        <Input
                            type="number"
                            value={yearFilter}
                            onChange={(e) => setYearFilter(e.target.value)}
                        />
                    </div>

                    <div>
                        <button type="submit" className="w-full h-10 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 transition-colors">
                            Apply Filters
                        </button>
                    </div>
                </form>

                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="px-6 py-4 font-semibold text-slate-600">Employee</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Type</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Dates</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Days</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Status</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Reason</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {loading ? (
                                <tr>
                                    <td colSpan={7} className="px-6 py-12 text-center text-slate-500">
                                        Loading requests...
                                    </td>
                                </tr>
                            ) : requests.length === 0 ? (
                                <tr>
                                    <td colSpan={7} className="px-6 py-12 text-center text-slate-500">
                                        No requests found matching criteria
                                    </td>
                                </tr>
                            ) : (
                                requests.map((req) => (
                                    <React.Fragment key={req.id}>
                                        <tr className="hover:bg-slate-50 transition-colors">
                                            <td className="px-6 py-4">
                                                <div>
                                                    <p className="font-medium text-slate-900">{req.user?.first_name} {req.user?.last_name}</p>
                                                    <p className="text-xs text-slate-500">{req.user?.email}</p>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 capitalize text-slate-700">
                                                {req.leave_type}
                                            </td>
                                            <td className="px-6 py-4 text-slate-600 whitespace-nowrap">
                                                <div className="flex flex-col">
                                                    <div className="flex items-center gap-2">
                                                        <span>{format(new Date(req.start_date), 'MMM d, yyyy')}</span>
                                                        {req.is_half_day && (
                                                            <span className="px-1.5 py-0.5 rounded text-[10px] uppercase font-semibold bg-brand-50 text-brand-700">
                                                                Half-Day {req.half_day_period}
                                                            </span>
                                                        )}
                                                    </div>
                                                    {!req.is_half_day && (
                                                        <span className="text-xs text-slate-400">to {format(new Date(req.end_date), 'MMM d, yyyy')}</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-slate-600">
                                                {formatDuration(getDisplayDuration(req.duration_days, req.start_date, req.end_date))}
                                            </td>
                                            <td className="px-6 py-4">
                                                <Badge variant={getStatusVariant(req.status)}>
                                                    {req.status}
                                                </Badge>
                                                {req.status === 'pending' && req.workflow_state?.current_step && (
                                                    <div className="mt-1">
                                                        <span className="text-[10px] uppercase tracking-wide font-semibold text-slate-500 bg-slate-100 px-1.5 py-0.5 rounded border border-slate-200">
                                                            Waiting: {req.workflow_state.current_step.responsible_role}
                                                        </span>
                                                    </div>
                                                )}
                                            </td>
                                            <td className="px-6 py-4 text-slate-600 max-w-xs" title={req.reason}>
                                                <div className="flex items-center gap-2">
                                                    <span className="truncate">{req.reason}</span>
                                                    {req.attachment_url && (
                                                        <a
                                                            href={req.attachment_url}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="text-brand-600 hover:text-brand-700 md:inline-flex items-center shrink-0 hidden"
                                                            title="View Attachment"
                                                        >
                                                            <FileText className="h-4 w-4" />
                                                        </a>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        size="sm"
                                                        variant="ghost"
                                                        className="h-8 w-8 p-0 rounded-full bg-purple-50 text-purple-600 hover:bg-purple-100"
                                                        onClick={() => setShowWorkflow(prev => ({ ...prev, [req.id]: !prev[req.id] }))}
                                                        title="View Workflow"
                                                    >
                                                        <GitBranch className="h-4 w-4" />
                                                    </Button>
                                                </div>
                                            </td>
                                        </tr>
                                        {showWorkflow[req.id] && (
                                            <tr className="bg-slate-50">
                                                <td colSpan={7} className="px-6 py-3">
                                                    <WorkflowStateDisplay
                                                        requestId={req.id}
                                                        applicantId={req.user_id}
                                                        currentStatus={req.status}
                                                        onActionComplete={fetchRequests}
                                                        showActions={req.status === 'pending' && canActionRequest(req)}
                                                        isNegativeBalance={req.is_negative_balance}
                                                    />
                                                </td>
                                            </tr>
                                        )}
                                    </React.Fragment>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </Card>
        </div>
    );
};

export default HRLeaves;
