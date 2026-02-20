import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Badge } from '../components/ui/Badge';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon, Info } from 'lucide-react';
import api from '../services/api';
import type { LeaveRequest } from '../types';

const DAYS_OF_WEEK = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

const getDaysInMonth = (year: number, month: number) => {
    return new Date(year, month + 1, 0).getDate();
};

const getFirstDayOfMonth = (year: number, month: number) => {
    return new Date(year, month, 1).getDay();
};

const TeamCalendar: React.FC = () => {
    const [currentDate, setCurrentDate] = useState(new Date());
    const [leaves, setLeaves] = useState<LeaveRequest[]>([]);
    const [loading, setLoading] = useState(false);
    const [selectedLeave, setSelectedLeave] = useState<LeaveRequest | null>(null);

    const currentYear = currentDate.getFullYear();
    const currentMonth = currentDate.getMonth();

    const fetchCalendarData = async () => {
        setLoading(true);
        try {
            // Month is 1-indexed for the API query
            const response = await api.get(`/team/calendar?year=${currentYear}&month=${currentMonth + 1}`);
            setLeaves(response.data || []);
        } catch (error) {
            console.error('Failed to fetch team calendar', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchCalendarData();
    }, [currentYear, currentMonth]);

    const handlePrevMonth = () => {
        setCurrentDate(new Date(currentYear, currentMonth - 1, 1));
    };

    const handleNextMonth = () => {
        setCurrentDate(new Date(currentYear, currentMonth + 1, 1));
    };

    const handleToday = () => {
        setCurrentDate(new Date());
    };

    const daysInMonth = getDaysInMonth(currentYear, currentMonth);
    const firstDay = getFirstDayOfMonth(currentYear, currentMonth);

    // Build grid cells
    const cells = [];
    // Padding for days before the 1st of the month
    for (let i = 0; i < firstDay; i++) {
        cells.push(<div key={`empty-${i}`} className="min-h-[100px] bg-slate-50 border border-slate-100 rounded-lg opacity-50"></div>);
    }

    // Actual days
    for (let day = 1; day <= daysInMonth; day++) {
        const currentCellDate = new Date(currentYear, currentMonth, day);
        // Reset time for accurate comparison
        currentCellDate.setHours(0, 0, 0, 0);

        // Find leaves that overlap with this day
        const dayLeaves = leaves.filter(leave => {
            const start = new Date(leave.start_date);
            const end = new Date(leave.end_date);
            start.setHours(0, 0, 0, 0);
            end.setHours(23, 59, 59, 999);
            return currentCellDate >= start && currentCellDate <= end;
        });

        const isToday = currentCellDate.toDateString() === new Date().toDateString();

        cells.push(
            <div
                key={day}
                className={`min-h-[100px] p-2 border rounded-lg transition-colors ${isToday ? 'bg-brand-50 border-brand-200' : 'bg-white border-slate-200'
                    }`}
            >
                <div className="flex justify-between items-start mb-2">
                    <span className={`text-sm font-semibold ${isToday ? 'text-brand-700' : 'text-slate-700'}`}>
                        {day}
                    </span>
                    {dayLeaves.length > 0 && (
                        <span className="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded-full">
                            {dayLeaves.length}
                        </span>
                    )}
                </div>
                <div className="space-y-1">
                    {dayLeaves.map(leave => {
                        const isApproved = leave.status === 'approved';
                        return (
                            <div
                                key={leave.id}
                                onClick={() => setSelectedLeave(leave)}
                                className={`text-[10px] sm:text-xs truncate px-1.5 py-1 rounded cursor-pointer border hover:opacity-80 transition-opacity ${isApproved
                                    ? 'bg-green-50 text-green-700 border-green-200'
                                    : 'bg-yellow-50 text-yellow-700 border-yellow-200'
                                    }`}
                                title={`${leave.user?.first_name} ${leave.user?.last_name} - ${leave.leave_type}`}
                            >
                                <span className="font-semibold">{leave.user?.first_name}</span> - {leave.leave_type.slice(0, 3).toUpperCase()}
                            </div>
                        );
                    })}
                </div>
            </div>
        );
    }

    return (
        <div className="max-w-6xl mx-auto space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Team Calendar</h1>
                    <p className="text-slate-500 mt-1">View your team's upcoming and ongoing leaves</p>
                </div>
            </div>

            <Card>
                <CardHeader className="flex flex-row items-center justify-between pb-4 border-b">
                    <div className="flex items-center gap-4">
                        <Button variant="outline" size="sm" onClick={handleToday}>Today</Button>
                        <div className="flex items-center gap-2">
                            <Button variant="ghost" size="sm" onClick={handlePrevMonth} className="px-2">
                                <ChevronLeft className="h-4 w-4" />
                            </Button>
                            <h2 className="text-lg font-semibold min-w-[140px] text-center">
                                {currentDate.toLocaleString('default', { month: 'long', year: 'numeric' })}
                            </h2>
                            <Button variant="ghost" size="sm" onClick={handleNextMonth} className="px-2">
                                <ChevronRight className="h-4 w-4" />
                            </Button>
                        </div>
                    </div>
                    {loading && (
                        <div className="text-sm text-slate-500 animate-pulse flex items-center gap-2">
                            <CalendarIcon className="h-4 w-4" /> Loading...
                        </div>
                    )}
                </CardHeader>
                <CardContent className="pt-6">
                    <div className="grid grid-cols-7 gap-2 mb-2">
                        {DAYS_OF_WEEK.map(day => (
                            <div key={day} className="text-center text-xs font-semibold text-slate-500 uppercase tracking-wider py-2">
                                {day}
                            </div>
                        ))}
                    </div>
                    <div className="grid grid-cols-7 gap-2">
                        {cells}
                    </div>
                </CardContent>
            </Card>

            {/* Leave Detail Modal Overlay */}
            {selectedLeave && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm">
                    <div className="bg-white rounded-xl shadow-xl max-w-md w-full overflow-hidden">
                        <div className="p-4 border-b border-slate-100 flex justify-between items-center bg-slate-50">
                            <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                                <Info className="h-4 w-4 text-brand-600" />
                                Leave Details
                            </h3>
                            <button onClick={() => setSelectedLeave(null)} className="text-slate-400 hover:text-slate-600">
                                ✕
                            </button>
                        </div>
                        <div className="p-6 space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <p className="font-semibold text-lg text-slate-900">
                                        {selectedLeave.user?.first_name} {selectedLeave.user?.last_name}
                                    </p>
                                    <p className="text-sm text-slate-500 capitalize">{selectedLeave.leave_type.replace('_', ' ')} Leave</p>
                                </div>
                                <Badge variant={selectedLeave.status === 'approved' ? 'success' : 'warning'}>
                                    {selectedLeave.status}
                                </Badge>
                            </div>

                            <div className="bg-slate-50 p-4 rounded-lg space-y-2 text-sm border border-slate-100">
                                <div className="flex justify-between">
                                    <span className="text-slate-500">Start Date:</span>
                                    <span className="font-medium">{new Date(selectedLeave.start_date).toLocaleDateString()}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-slate-500">End Date:</span>
                                    <span className="font-medium">{new Date(selectedLeave.end_date).toLocaleDateString()}</span>
                                </div>
                                <div className="flex justify-between border-t border-slate-200 pt-2 mt-2">
                                    <span className="text-slate-500">Duration:</span>
                                    <span className="font-medium">{selectedLeave.duration_days} days</span>
                                </div>
                            </div>

                            {selectedLeave.reason && (
                                <div>
                                    <span className="text-sm font-medium text-slate-700">Reason</span>
                                    <p className="text-sm text-slate-600 mt-1 bg-white border border-slate-200 p-3 rounded-lg leading-relaxed">
                                        {selectedLeave.reason}
                                    </p>
                                </div>
                            )}
                        </div>
                        <div className="p-4 bg-slate-50 flex justify-end border-t border-slate-100">
                            <Button onClick={() => setSelectedLeave(null)}>Close</Button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default TeamCalendar;
