import React, { useEffect, useState } from 'react';
import { Download, FileText, Calendar, TrendingUp, CheckCircle, XCircle, Clock } from 'lucide-react';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import api from '../services/api';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import type { LeaveRequest } from '../types';

interface DashboardStats {
    status_counts: { Status: string; Count: number }[];
    type_counts: { LeaveType: string; Count: number }[];
    recent_activity: LeaveRequest[];
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884d8'];

const Reports: React.FC = () => {
    const [month, setMonth] = useState(new Date().getMonth() + 1);
    const [year, setYear] = useState(new Date().getFullYear());
    const [loading, setLoading] = useState(false);
    const [statsLoading, setStatsLoading] = useState(true);
    const [stats, setStats] = useState<DashboardStats | null>(null);

    const fetchStats = async () => {
        setStatsLoading(true);
        try {
            const response = await api.get('/admin/dashboard-stats');
            setStats(response.data);
        } catch (error) {
            console.error("Failed to fetch dashboard stats", error);
        } finally {
            setStatsLoading(false);
        }
    };

    useEffect(() => {
        fetchStats();
    }, []);

    const handleDownload = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            const response = await api.get('/hr/payroll-report', {
                params: { month, year },
                responseType: 'blob'
            });

            // Create download link
            const url = window.URL.createObjectURL(new Blob([response.data]));
            const link = document.createElement('a');
            link.href = url;
            link.setAttribute('download', `payroll_report_${year}_${month}.csv`);
            document.body.appendChild(link);
            link.click();
            link.parentNode?.removeChild(link);
        } catch (error) {
            console.error("Failed to download report", error);
            alert("Failed to download report. Please try again.");
        } finally {
            setLoading(false);
        }
    };

    // Helper to get count by status
    const getCount = (status: string) => {
        if (!stats) return 0;
        const item = stats.status_counts.find(s => s.Status === status);
        return item ? item.Count : 0;
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Reports & Analytics</h1>
                    <p className="text-slate-500 mt-1">System-wide metrics and exportable reports</p>
                </div>
                <Button variant="outline" onClick={fetchStats} isLoading={statsLoading}>
                    Refresh Data
                </Button>
            </div>

            {/* Overview Stats Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <Card>
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-blue-100 text-blue-600 rounded-full">
                            <FileText className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-slate-500">Total Requests</p>
                            <h3 className="text-2xl font-bold text-slate-900">
                                {stats?.status_counts.reduce((acc, curr) => acc + curr.Count, 0) || 0}
                            </h3>
                        </div>
                    </CardContent>
                </Card>

                <Card>
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-amber-100 text-amber-600 rounded-full">
                            <Clock className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-slate-500">Pending Action</p>
                            <h3 className="text-2xl font-bold text-slate-900">{getCount('pending')}</h3>
                        </div>
                    </CardContent>
                </Card>

                <Card>
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-green-100 text-green-600 rounded-full">
                            <CheckCircle className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-slate-500">Approved</p>
                            <h3 className="text-2xl font-bold text-slate-900">{getCount('approved')}</h3>
                        </div>
                    </CardContent>
                </Card>

                <Card>
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-red-100 text-red-600 rounded-full">
                            <XCircle className="h-6 w-6" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-slate-500">Rejected</p>
                            <h3 className="text-2xl font-bold text-slate-900">{getCount('rejected')}</h3>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Charts Section */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <TrendingUp className="h-5 w-5 text-slate-500" />
                            Approved Leaves by Type
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="h-80">
                        {stats && stats.type_counts.length > 0 ? (
                            <ResponsiveContainer width="100%" height="100%">
                                <PieChart>
                                    <Pie
                                        data={stats.type_counts}
                                        cx="50%"
                                        cy="50%"
                                        labelLine={false}
                                        label={({ name, percent }: { name?: string, percent?: number }) => `${name || ''} ${((percent || 0) * 100).toFixed(0)}%`}
                                        outerRadius={80}
                                        fill="#8884d8"
                                        dataKey="Count"
                                        nameKey="LeaveType"
                                    >
                                        {stats.type_counts.map((_, index) => (
                                            <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                        ))}
                                    </Pie>
                                    <Tooltip />
                                    <Legend />
                                </PieChart>
                            </ResponsiveContainer>
                        ) : (
                            <div className="flex items-center justify-center h-full text-slate-400">
                                No approved leaves data available
                            </div>
                        )}
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <Clock className="h-5 w-5 text-slate-500" />
                            Recent Activity
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            {stats?.recent_activity.map((req) => (
                                <div key={req.id} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg border border-slate-100">
                                    <div className="flex items-center gap-3">
                                        <div className={`w-2 h-2 rounded-full ${req.status === 'approved' ? 'bg-green-500' :
                                            req.status === 'rejected' ? 'bg-red-500' :
                                                req.status === 'pending' ? 'bg-amber-500' : 'bg-slate-300'
                                            }`} />
                                        <div>
                                            <p className="text-sm font-medium text-slate-900">
                                                {req.user?.first_name} {req.user?.last_name}
                                            </p>
                                            <p className="text-xs text-slate-500">
                                                Requested {req.leave_type} leave
                                            </p>
                                        </div>
                                    </div>
                                    <div className="text-right">
                                        <span className={`text-xs font-medium px-2 py-1 rounded-full ${req.status === 'approved' ? 'bg-green-100 text-green-700' :
                                            req.status === 'rejected' ? 'bg-red-100 text-red-700' :
                                                req.status === 'pending' ? 'bg-amber-100 text-amber-700' : 'bg-slate-100 text-slate-700'
                                            }`}>
                                            {req.status}
                                        </span>
                                        <p className="text-xs text-slate-400 mt-1">
                                            {new Date(req.created_at).toLocaleDateString()}
                                        </p>
                                    </div>
                                </div>
                            ))}
                            {(!stats?.recent_activity || stats.recent_activity.length === 0) && (
                                <p className="text-center text-slate-400 py-4">No recent activity</p>
                            )}
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Export Section */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <Card className="p-6">
                    <div className="flex items-start gap-4">
                        <div className="p-3 bg-brand-100 rounded-lg text-brand-600">
                            <FileText className="h-6 w-6" />
                        </div>
                        <div className="flex-1">
                            <h3 className="text-lg font-semibold text-slate-900 mb-1">Payroll Report</h3>
                            <p className="text-slate-500 mb-6 text-sm">
                                Export monthly payroll data including leave balances, unpaid leaves, and attendance summary.
                            </p>

                            <form onSubmit={handleDownload} className="space-y-4">
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-1">
                                        <label className="text-sm font-medium text-slate-700">Month</label>
                                        <select
                                            className="w-full h-10 px-3 rounded-lg border border-slate-300 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all"
                                            value={month}
                                            onChange={(e) => setMonth(parseInt(e.target.value))}
                                        >
                                            {Array.from({ length: 12 }, (_, i) => i + 1).map(m => (
                                                <option key={m} value={m}>
                                                    {new Date(0, m - 1).toLocaleString('default', { month: 'long' })}
                                                </option>
                                            ))}
                                        </select>
                                    </div>
                                    <div className="space-y-1">
                                        <label className="text-sm font-medium text-slate-700">Year</label>
                                        <Input
                                            type="number"
                                            value={year}
                                            onChange={(e) => setYear(parseInt(e.target.value))}
                                            min={2000}
                                            max={2100}
                                        />
                                    </div>
                                </div>

                                <Button type="submit" className="w-full" isLoading={loading}>
                                    <Download className="h-4 w-4 mr-2" />
                                    Download CSV
                                </Button>
                            </form>
                        </div>
                    </div>
                </Card>

                <Card className="p-6 opacity-75 relative overflow-hidden">
                    <div className="absolute inset-0 bg-slate-50/50 backdrop-blur-[1px] flex items-center justify-center z-10">
                        <span className="bg-white px-3 py-1 rounded-full text-xs font-medium text-slate-500 shadow-sm border border-slate-200">Coming Soon</span>
                    </div>
                    <div className="flex items-start gap-4">
                        <div className="p-3 bg-indigo-100 rounded-lg text-indigo-600">
                            <Calendar className="h-6 w-6" />
                        </div>
                        <div className="flex-1">
                            <h3 className="text-lg font-semibold text-slate-900 mb-1">Attendance Summary</h3>
                            <p className="text-slate-500 mb-4 text-sm">
                                Detailed breakdown of employee attendance, late logins, and early logouts.
                            </p>
                            <Button className="w-full" disabled>
                                <Download className="h-4 w-4 mr-2" />
                                Download PDF
                            </Button>
                        </div>
                    </div>
                </Card>
            </div>
        </div>
    );
};

export default Reports;
