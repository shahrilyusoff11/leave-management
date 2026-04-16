import React, { useState, useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import type { User, UserDelegation } from '../types';
import { format } from 'date-fns';
import { Calendar, User as UserIcon, X, Search, Plus, Trash2 } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { useToast } from '../components/ui/Toast';
import { Modal } from '../components/ui/Modal';
import { ConfirmationModal } from '../components/ui/ConfirmationModal';
import { useAuth } from '../context/AuthContext';
import { canManageTeam } from '../utils/roles';

const Delegations: React.FC = () => {
    const { user } = useAuth();
    const [delegations, setDelegations] = useState<UserDelegation[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const { showToast } = useToast();

    // Form State
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');
    const [reason, setReason] = useState('');
    const [selectedDelegate, setSelectedDelegate] = useState<User | null>(null);

    // Search State
    const [query, setQuery] = useState('');
    const [candidates, setCandidates] = useState<User[]>([]);
    const [searching, setSearching] = useState(false);

    // Cancel State
    const [cancelId, setCancelId] = useState<string | null>(null);

    if (!canManageTeam(user?.role)) {
        return <Navigate to="/dashboard" replace />;
    }

    useEffect(() => {
        fetchDelegations();
    }, []);

    // Debounce search
    useEffect(() => {
        const timer = setTimeout(() => {
            if (query.length >= 2) {
                searchCandidates(query);
            } else {
                setCandidates([]);
            }
        }, 300);
        return () => clearTimeout(timer);
    }, [query]);

    const fetchDelegations = async () => {
        try {
            const token = localStorage.getItem('token');
            const response = await fetch('/api/v1/delegations', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (response.ok) {
                const data = await response.json();
                setDelegations(data);
            }
        } catch (error) {
            console.error('Failed to fetch delegations', error);
        } finally {
            setLoading(false);
        }
    };

    const searchCandidates = async (q: string) => {
        setSearching(true);
        try {
            const token = localStorage.getItem('token');
            const response = await fetch(`/api/v1/delegations/candidates?q=${q}`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (response.ok) {
                const data = await response.json();
                setCandidates(data);
            }
        } catch (error) {
            console.error('Search failed', error);
        } finally {
            setSearching(false);
        }
    };

    const handleCreate = async () => {
        if (!selectedDelegate || !startDate || !endDate) return;

        try {
            const token = localStorage.getItem('token');
            const response = await fetch('/api/v1/delegations', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    delegate_id: selectedDelegate.id,
                    start_date: new Date(startDate).toISOString(),
                    end_date: new Date(endDate).toISOString(),
                    reason
                })
            });

            if (response.ok) {
                showToast('Delegation created successfully', 'success');
                setShowAddModal(false);
                fetchDelegations();
                resetForm();
            } else {
                const err = await response.json();
                showToast(err.error || 'Failed to create delegation', 'error');
            }
        } catch (error) {
            showToast('Network error', 'error');
        }
    };

    const handleCancel = async () => {
        if (!cancelId) return;
        try {
            const token = localStorage.getItem('token');
            const response = await fetch(`/api/v1/delegations/${cancelId}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            });

            if (response.ok) {
                showToast('Delegation cancelled', 'success');
                fetchDelegations();
            } else {
                showToast('Failed to cancel', 'error');
            }
        } catch (error) {
            showToast('Network error', 'error');
        } finally {
            setCancelId(null);
        }
    };

    const resetForm = () => {
        setStartDate('');
        setEndDate('');
        setReason('');
        setSelectedDelegate(null);
        setQuery('');
        setCandidates([]);
    };

    return (
        <div className="container mx-auto px-4 py-8">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">My Delegations</h1>
                    <p className="text-slate-600">Assign an acting manager when you are unavailable.</p>
                </div>
                <Button onClick={() => setShowAddModal(true)}>
                    <Plus size={18} className="mr-2" />
                    Add Delegation
                </Button>
            </div>

            {loading ? (
                <div className="text-center py-10">Loading...</div>
            ) : delegations.length === 0 ? (
                <div className="bg-white rounded-lg shadow-sm p-10 text-center border border-slate-100">
                    <UserIcon className="mx-auto h-12 w-12 text-slate-300 mb-4" />
                    <h3 className="text-lg font-medium text-slate-900">No delegations found</h3>
                    <p className="text-slate-500 mt-1">You haven't delegated your responsibilities to anyone yet.</p>
                </div>
            ) : (
                <div className="grid gap-4">
                    {delegations.map((d) => (
                        <Card key={d.id} className="border-l-4 border-l-brand-500">
                            <div className="p-6 flex justify-between items-start">
                                <div>
                                    <div className="flex items-center gap-2 mb-1">
                                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${d.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800'
                                            }`}>
                                            {d.status.toUpperCase()}
                                        </span>
                                        <span className="text-sm text-slate-500 flex items-center gap-1">
                                            <Calendar size={14} />
                                            {format(new Date(d.start_date), 'MMM d, yyyy')} - {format(new Date(d.end_date), 'MMM d, yyyy')}
                                        </span>
                                    </div>
                                    <h3 className="text-lg font-semibold text-slate-900">
                                        Delegated to: {d.delegate ? `${d.delegate.first_name} ${d.delegate.last_name}` : 'Unknown'}
                                    </h3>
                                    {d.reason && (
                                        <p className="text-slate-600 mt-1 text-sm">"{d.reason}"</p>
                                    )}
                                </div>
                                {d.status === 'active' && (
                                    <button
                                        onClick={() => setCancelId(d.id)}
                                        className="text-slate-400 hover:text-red-500 transition-colors p-2"
                                        title="Cancel Delegation"
                                    >
                                        <Trash2 size={18} />
                                    </button>
                                )}
                            </div>
                        </Card>
                    ))}
                </div>
            )}

            <Modal
                isOpen={showAddModal}
                onClose={() => { setShowAddModal(false); resetForm(); }}
                title="Add New Delegation"
            >
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">Select Delegate</label>
                        {selectedDelegate ? (
                            <div className="flex items-center justify-between p-3 bg-brand-50 border border-brand-200 rounded-lg">
                                <span className="font-medium text-brand-900">
                                    {selectedDelegate.first_name} {selectedDelegate.last_name} ({selectedDelegate.email})
                                </span>
                                <button onClick={() => setSelectedDelegate(null)} className="text-brand-500 hover:text-brand-700">
                                    <X size={18} />
                                </button>
                            </div>
                        ) : (
                            <div className="relative">
                                <Search className="absolute left-3 top-3 text-slate-400" size={18} />
                                <input
                                    type="text"
                                    placeholder="Search by name or email..."
                                    className="w-full pl-10 pr-4 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500 outline-none transition-all"
                                    value={query}
                                    onChange={(e) => setQuery(e.target.value)}
                                />
                                {candidates.length > 0 && (
                                    <div className="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                                        {candidates.map(u => (
                                            <div
                                                key={u.id}
                                                className="p-3 hover:bg-slate-50 cursor-pointer flex justify-between items-center border-b border-slate-50 last:border-0"
                                                onClick={() => { setSelectedDelegate(u); setQuery(''); setCandidates([]); }}
                                            >
                                                <div>
                                                    <div className="font-medium text-slate-900">{u.first_name} {u.last_name}</div>
                                                    <div className="text-xs text-slate-500">{u.email}</div>
                                                </div>
                                                <div className="text-xs text-slate-400">{u.position || u.role}</div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                                {candidates.length === 0 && query.length >= 2 && !searching && (
                                    <div className="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg p-3 text-center text-slate-500 text-sm">
                                        No users found
                                    </div>
                                )}
                            </div>
                        )}
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <Input
                            label="Start Date"
                            type="date"
                            value={startDate}
                            onChange={(e) => setStartDate(e.target.value)}
                            required
                        />
                        <Input
                            label="End Date"
                            type="date"
                            value={endDate}
                            onChange={(e) => setEndDate(e.target.value)}
                            required
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">Reason (Optional)</label>
                        <textarea
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 outline-none transition-all"
                            rows={3}
                            value={reason}
                            onChange={(e) => setReason(e.target.value)}
                            placeholder="e.g. Annual Leave, Business Trip"
                        />
                    </div>

                    <div className="flex justify-end gap-3 mt-6">
                        <Button variant="ghost" onClick={() => setShowAddModal(false)}>Cancel</Button>
                        <Button onClick={handleCreate} disabled={!selectedDelegate || !startDate || !endDate}>
                            Create Delegation
                        </Button>
                    </div>
                </div>
            </Modal>

            <ConfirmationModal
                isOpen={!!cancelId}
                onClose={() => setCancelId(null)}
                onConfirm={handleCancel}
                title="Cancel Delegation"
                message="Are you sure you want to cancel this delegation? This action cannot be undone."
                confirmText="Yes, Cancel"
                cancelText="No, Keep It"
                type="danger"
            />
        </div>
    );
};

export default Delegations;
