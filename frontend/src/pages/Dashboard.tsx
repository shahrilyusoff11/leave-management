import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Calendar, CheckCircle, Clock, GitBranch } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import api from '../services/api';
import type { LeaveBalance, LeaveRequest } from '../types';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Link } from 'react-router-dom';
import { getDisplayDuration, formatDuration } from '../utils/duration';
import WorkflowStateDisplay from '../components/WorkflowStateDisplay';

const Dashboard: React.FC = () => {
    const { user } = useAuth();
    const [balances, setBalances] = useState<LeaveBalance[]>([]);
    const [recentRequests, setRecentRequests] = useState<LeaveRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [showWorkflow, setShowWorkflow] = useState<Record<string, boolean>>({});

    const fetchData = async () => {
        try {
            const [balanceRes, requestsRes] = await Promise.all([
                api.get('/leave-balance'),
                api.get('/leave-requests')
            ]);

            // Handle balance response - can be object keyed by leave type or array
            const balanceData = balanceRes.data;
            if (balanceData && typeof balanceData === 'object' && !Array.isArray(balanceData)) {
                // Convert object format to array format
                const balanceArray = Object.entries(balanceData).map(([leaveType, data]: [string, any]) => ({
                    leave_type: leaveType,
                    total_entitlement: data.total_entitlement || 0,
                    used: data.used || 0,
                    carried_forward: data.carried_forward || 0,
                    adjusted: data.adjusted || 0,
                    available: data.available || 0,
                })) as unknown as LeaveBalance[];
                setBalances(balanceArray);
            } else {
                setBalances(Array.isArray(balanceData) ? balanceData : []);
            }
            setRecentRequests(Array.isArray(requestsRes.data) ? requestsRes.data.slice(0, 5) : []);
        } catch (error) {
            console.error("Failed to fetch dashboard data", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
    }, []);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'approved': return 'success';
            case 'rejected': return 'danger';
            case 'pending': return 'warning';
            default: return 'default';
        }
    };


    if (loading) {
        return <div className="p-8 text-center text-slate-500">Loading dashboard...</div>;
    }

    // Calculate remaining days for standard types
    const annual = balances.find(b => b.leave_type === 'annual');
    const sick = balances.find(b => b.leave_type === 'sick');

    const getRemaining = (b?: LeaveBalance) => {
        if (!b) return 0;
        // Use available if already calculated, otherwise calculate
        if ((b as any).available !== undefined) return (b as any).available;
        return b.total_entitlement - b.used + b.carried_forward + ((b as any).adjusted || 0);
    };

    const formatLeaveDays = (value: number) => {
        const roundedToHalfDay = Math.round(value * 2) / 2;

        if (Number.isInteger(roundedToHalfDay)) {
            return roundedToHalfDay.toString();
        }

        return roundedToHalfDay.toFixed(1);
    };

    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-3xl font-bold text-slate-900">Dashboard</h1>
                <p className="text-slate-500 mt-2">Welcome back, {user?.first_name}!</p>
            </div>

            {/* Stats Grid */}
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {/* Annual Leave */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3 }}
                >
                    <Card className="border-l-4 border-l-brand-500 overflow-hidden relative">
                        <div className="absolute top-0 right-0 p-4 opacity-10">
                            <Calendar className="h-24 w-24 text-brand-600" />
                        </div>
                        <CardHeader className="pb-2">
                            <CardTitle className="text-sm font-medium text-slate-500 uppercase tracking-wider">Annual Leave</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="text-4xl font-bold text-slate-900">{formatLeaveDays(getRemaining(annual))}</div>
                            <p className="text-sm text-slate-500 mt-1">days remaining</p>
                            <div className="mt-4 h-1.5 w-full bg-slate-100 rounded-full overflow-hidden">
                                <div
                                    className="h-full bg-brand-500 rounded-full"
                                    style={{
                                        width: `${Math.min(100, (annual?.used || 0) / ((annual?.total_entitlement || 0) + (annual?.carried_forward || 0) + ((annual as any)?.adjusted || 0) || 1) * 100)}%`
                                    }}
                                />
                            </div>
                        </CardContent>
                    </Card>
                </motion.div>

                {/* Sick Leave */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: 0.1 }}
                >
                    <Card className="border-l-4 border-l-emerald-500 overflow-hidden relative">
                        <div className="absolute top-0 right-0 p-4 opacity-10">
                            <CheckCircle className="h-24 w-24 text-emerald-600" />
                        </div>
                        <CardHeader className="pb-2">
                            <CardTitle className="text-sm font-medium text-slate-500 uppercase tracking-wider">Sick Leave</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="text-4xl font-bold text-slate-900">{formatLeaveDays(getRemaining(sick))}</div>
                            <p className="text-sm text-slate-500 mt-1">days remaining</p>
                            <div className="mt-4 h-1.5 w-full bg-slate-100 rounded-full overflow-hidden">
                                <div
                                    className="h-full bg-emerald-500 rounded-full"
                                    style={{
                                        width: `${Math.min(100, (sick?.used || 0) / ((sick?.total_entitlement || 0) + (sick?.carried_forward || 0) + ((sick as any)?.adjusted || 0) || 1) * 100)}%`
                                    }}
                                />
                            </div>
                        </CardContent>
                    </Card>
                </motion.div>

                {/* Pending Requests */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: 0.2 }}
                >
                    <Card className="border-l-4 border-l-amber-500 overflow-hidden relative">
                        <div className="absolute top-0 right-0 p-4 opacity-10">
                            <Clock className="h-24 w-24 text-amber-600" />
                        </div>
                        <CardHeader className="pb-2">
                            <CardTitle className="text-sm font-medium text-slate-500 uppercase tracking-wider">Pending Requests</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <div className="text-4xl font-bold text-slate-900">
                                {recentRequests.filter(r => r.status === 'pending').length}
                            </div>
                            <p className="text-sm text-slate-500 mt-1">requests awaiting approval</p>
                            <div className="mt-4">
                                <Link to="/request-leave">
                                    <Button variant="outline" size="sm" className="w-full text-xs">New Request</Button>
                                </Link>
                            </div>
                        </CardContent>
                    </Card>
                </motion.div>
            </div>

            {/* Recent Activity */}
            <div>
                <h2 className="text-lg font-bold text-slate-900 mb-4">Recent Requests</h2>
                <Card className="overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                            <thead>
                                <tr className="bg-slate-50 border-b border-slate-200">
                                    <th className="px-6 py-4 font-semibold text-slate-600">Type</th>
                                    <th className="px-6 py-4 font-semibold text-slate-600">Dates</th>
                                    <th className="px-6 py-4 font-semibold text-slate-600">Duration</th>
                                    <th className="px-6 py-4 font-semibold text-slate-600">Status</th>
                                    <th className="px-6 py-4 font-semibold text-slate-600">Date Applied</th>
                                    <th className="px-6 py-4 font-semibold text-slate-600 text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100">
                                {recentRequests.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-6 py-8 text-center text-slate-500">
                                            No leave requests found.
                                        </td>
                                    </tr>
                                ) : (
                                    recentRequests.map((req) => (
                                        <React.Fragment key={req.id}>
                                            <tr className="hover:bg-slate-50 transition-colors">
                                                <td className="px-6 py-4 font-medium text-slate-900 capitalize text-base">
                                                    {req.leave_type}
                                                </td>
                                                <td className="px-6 py-4 text-slate-600">
                                                    {new Date(req.start_date).toLocaleDateString()} - {new Date(req.end_date).toLocaleDateString()}
                                                </td>
                                                <td className="px-6 py-4 text-slate-600">
                                                    {formatDuration(getDisplayDuration(req.duration_days, req.start_date, req.end_date))}
                                                </td>
                                                <td className="px-6 py-4">
                                                    <Badge variant={getStatusColor(req.status) as any}>
                                                        {req.status}
                                                    </Badge>
                                                </td>
                                                <td className="px-6 py-4 text-slate-400 text-xs">
                                                    {new Date(req.created_at).toLocaleDateString()}
                                                </td>
                                                <td className="px-6 py-4 text-right">
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="h-8 w-8 p-0 rounded-full bg-purple-50 text-purple-600 hover:bg-purple-100"
                                                        onClick={() => setShowWorkflow(prev => ({ ...prev, [req.id]: !prev[req.id] }))}
                                                        title="View Workflow"
                                                    >
                                                        <GitBranch className="h-4 w-4" />
                                                    </Button>
                                                </td>
                                            </tr>
                                            {showWorkflow[req.id] && (
                                                <tr className="bg-slate-50">
                                                    <td colSpan={6} className="px-6 py-3">
                                                        <WorkflowStateDisplay
                                                            requestId={req.id}
                                                            applicantId={req.user_id}
                                                            currentStatus={req.status}
                                                            onActionComplete={fetchData}
                                                            showActions={req.status === 'pending'} // Dashboard usually shows actionables
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
                    <div className="p-4 border-t border-slate-100 flex justify-center">
                        <Link to="/my-leaves">
                            <Button variant="ghost" size="sm">View All History</Button>
                        </Link>
                    </div>
                </Card>
            </div>
        </div>
    );
};

export default Dashboard;
