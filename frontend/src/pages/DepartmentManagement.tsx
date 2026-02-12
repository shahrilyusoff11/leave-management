import React, { useEffect, useState } from 'react';
import { Plus, Trash2, Building2, Users, ArrowRightLeft } from 'lucide-react';
import api from '../services/api';
import type { User, UserRole, Department, HODDelegation } from '../types';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { useToast } from '../components/ui/Toast';
import { getRoleLabel } from '../utils/roles';
import { format } from 'date-fns';

const DepartmentManagement: React.FC = () => {
    const { showToast } = useToast();
    const [departments, setDepartments] = useState<Department[]>([]);
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const [editDept, setEditDept] = useState<Department | null>(null);
    const [delegationDept, setDelegationDept] = useState<Department | null>(null);
    const [delegations, setDelegations] = useState<HODDelegation[]>([]);

    const fetchAll = async () => {
        setLoading(true);
        try {
            const [deptsRes, usersRes] = await Promise.all([
                api.get('/hr/departments'),
                api.get('/hr/users')
            ]);
            if (Array.isArray(deptsRes.data)) setDepartments(deptsRes.data);
            if (Array.isArray(usersRes.data)) setUsers(usersRes.data);
        } catch (err) {
            console.error('Failed to fetch departments', err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { fetchAll(); }, []);

    const fetchDelegations = async (deptId: string) => {
        try {
            const res = await api.get(`/hr/departments/${deptId}/delegations`);
            if (Array.isArray(res.data)) setDelegations(res.data);
            else setDelegations([]);
        } catch {
            setDelegations([]);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this department?')) return;
        try {
            await api.delete(`/hr/departments/${id}`);
            showToast('Department deleted', 'success');
            fetchAll();
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to delete', 'error');
        }
    };

    const handleDeleteDelegation = async (delegationId: string) => {
        try {
            await api.delete(`/hr/departments/delegations/${delegationId}`);
            showToast('Delegation removed', 'success');
            if (delegationDept) fetchDelegations(delegationDept.id);
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to remove delegation', 'error');
        }
    };

    const getUserName = (id?: string) => {
        if (!id) return '—';
        const u = users.find(u => u.id === id);
        return u ? `${u.first_name} ${u.last_name}` : '—';
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Department Management</h1>
                    <p className="text-slate-500 mt-1">Manage departments, HOD assignments, and acting HOD delegations</p>
                </div>
                <Button onClick={() => setIsCreateOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Department
                </Button>
            </div>

            {loading ? (
                <div className="text-center py-12 text-slate-500">Loading departments...</div>
            ) : departments.length === 0 ? (
                <Card className="p-12 text-center">
                    <Building2 className="h-12 w-12 mx-auto text-slate-300 mb-3" />
                    <p className="text-slate-500">No departments created yet</p>
                    <Button className="mt-4" onClick={() => setIsCreateOpen(true)}>Create First Department</Button>
                </Card>
            ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {departments.map(dept => (
                        <Card key={dept.id} className="p-5 space-y-4 hover:shadow-md transition-shadow">
                            <div className="flex items-start justify-between">
                                <div className="flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-lg bg-brand-50 flex items-center justify-center">
                                        <Building2 className="h-5 w-5 text-brand-600" />
                                    </div>
                                    <div>
                                        <h3 className="font-semibold text-slate-900">{dept.name}</h3>
                                        <p className="text-xs text-slate-400">
                                            {users.filter(u => u.department_id === dept.id).length} members
                                        </p>
                                    </div>
                                </div>
                                <div className="flex gap-1">
                                    <button
                                        onClick={() => setEditDept(dept)}
                                        className="p-1.5 rounded-md hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition-colors"
                                        title="Edit"
                                    >
                                        <Users className="h-4 w-4" />
                                    </button>
                                    <button
                                        onClick={() => handleDelete(dept.id)}
                                        className="p-1.5 rounded-md hover:bg-red-50 text-slate-400 hover:text-red-500 transition-colors"
                                        title="Delete"
                                    >
                                        <Trash2 className="h-4 w-4" />
                                    </button>
                                </div>
                            </div>

                            <div className="border-t border-slate-100 pt-3 space-y-2">
                                <div className="flex items-center justify-between text-sm">
                                    <span className="text-slate-500">Head of Department</span>
                                    <span className="font-medium text-slate-700">{getUserName(dept.hod_id)}</span>
                                </div>
                            </div>

                            <button
                                onClick={() => { setDelegationDept(dept); fetchDelegations(dept.id); }}
                                className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-lg border border-slate-200 text-sm text-slate-600 hover:bg-slate-50 hover:border-slate-300 transition-colors"
                            >
                                <ArrowRightLeft className="h-3.5 w-3.5" />
                                Manage HOD Delegations
                            </button>
                        </Card>
                    ))}
                </div>
            )}

            {/* Create/Edit Department Modal */}
            <DeptFormModal
                isOpen={isCreateOpen || !!editDept}
                dept={editDept}
                users={users}
                onClose={() => { setIsCreateOpen(false); setEditDept(null); }}
                onSuccess={() => { fetchAll(); setIsCreateOpen(false); setEditDept(null); }}
            />

            {/* Delegation Modal */}
            {delegationDept && (
                <DelegationModal
                    isOpen={!!delegationDept}
                    dept={delegationDept}
                    users={users}
                    delegations={delegations}
                    onClose={() => setDelegationDept(null)}
                    onDeleteDelegation={handleDeleteDelegation}
                    onCreateDelegation={async () => { if (delegationDept) fetchDelegations(delegationDept.id); }}
                />
            )}
        </div>
    );
};

// Department Create/Edit Form Modal
const DeptFormModal = ({ isOpen, dept, users, onClose, onSuccess }: {
    isOpen: boolean; dept: Department | null; users: User[]; onClose: () => void; onSuccess: () => void;
}) => {
    const { showToast } = useToast();
    const [name, setName] = useState('');
    const [hodId, setHodId] = useState('');
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (dept) {
            setName(dept.name);
            setHodId(dept.hod_id || '');
        } else {
            setName('');
            setHodId('');
        }
    }, [dept, isOpen]);

    const handleSave = async () => {
        if (!name.trim()) return;
        setSaving(true);
        try {
            const payload = { name, hod_id: hodId || null };
            if (dept) {
                await api.put(`/hr/departments/${dept.id}`, payload);
                showToast('Department updated', 'success');
            } else {
                await api.post('/hr/departments', payload);
                showToast('Department created', 'success');
            }
            onSuccess();
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to save', 'error');
        } finally {
            setSaving(false);
        }
    };

    // Filter to HOD-eligible users
    const hodCandidates = users.filter(u =>
        ['hod', 'manager', 'admin', 'sysadmin'].includes(u.role) && u.is_active !== false
    );

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={dept ? 'Edit Department' : 'Create Department'}>
            <div className="space-y-4">
                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Department Name</label>
                    <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Engineering" />
                </div>
                <div className="space-y-1">
                    <label className="text-sm font-medium text-slate-700">Head of Department</label>
                    <select
                        value={hodId}
                        onChange={(e) => setHodId(e.target.value)}
                        className="w-full h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                    >
                        <option value="">No HOD Assigned</option>
                        {hodCandidates.map(u => (
                            <option key={u.id} value={u.id}>
                                {u.first_name} {u.last_name} — {getRoleLabel(u.role as UserRole)}
                            </option>
                        ))}
                    </select>
                    <p className="text-xs text-slate-400">The HOD will receive approval requests from department members</p>
                </div>
                <div className="flex justify-end gap-3 mt-6">
                    <Button variant="ghost" onClick={onClose}>Cancel</Button>
                    <Button onClick={handleSave} isLoading={saving}>
                        {dept ? 'Save Changes' : 'Create Department'}
                    </Button>
                </div>
            </div>
        </Modal>
    );
};

// HOD Delegation Modal
const DelegationModal = ({ isOpen, dept, users, delegations, onClose, onDeleteDelegation, onCreateDelegation }: {
    isOpen: boolean; dept: Department; users: User[]; delegations: HODDelegation[];
    onClose: () => void; onDeleteDelegation: (id: string) => void; onCreateDelegation: () => void;
}) => {
    const { showToast } = useToast();
    const [delegateId, setDelegateId] = useState('');
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');
    const [reason, setReason] = useState('');
    const [saving, setSaving] = useState(false);

    const handleCreate = async () => {
        if (!delegateId || !startDate || !endDate) {
            showToast('Please fill all required fields', 'error');
            return;
        }
        setSaving(true);
        try {
            await api.post(`/hr/departments/${dept.id}/delegations`, {
                delegator_id: dept.hod_id,
                delegate_id: delegateId,
                start_date: startDate,
                end_date: endDate,
                reason
            });
            showToast('Delegation created', 'success');
            setDelegateId('');
            setStartDate('');
            setEndDate('');
            setReason('');
            onCreateDelegation();
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to create delegation', 'error');
        } finally {
            setSaving(false);
        }
    };

    const eligibleDelegates = users.filter(u =>
        u.is_active !== false && u.id !== dept.hod_id
    );

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={`HOD Delegations — ${dept.name}`} className="max-w-2xl" position="top">
            <div className="space-y-6">
                {/* Existing Delegations */}
                <div>
                    <h4 className="text-sm font-semibold text-slate-700 mb-3">Active & Scheduled Delegations</h4>
                    {delegations.length === 0 ? (
                        <div className="text-center py-6 bg-slate-50 rounded-lg text-sm text-slate-500">
                            No delegations scheduled
                        </div>
                    ) : (
                        <div className="space-y-2">
                            {delegations.map(d => {
                                const isActive = d.is_active && new Date(d.start_date) <= new Date() && new Date(d.end_date) >= new Date();
                                return (
                                    <div key={d.id} className={`flex items-center justify-between p-3 rounded-lg border ${isActive ? 'border-green-200 bg-green-50' : 'border-slate-200 bg-white'}`}>
                                        <div>
                                            <div className="flex items-center gap-2">
                                                <span className="font-medium text-sm text-slate-900">
                                                    {d.delegate ? `${d.delegate.first_name} ${d.delegate.last_name}` : delegateId}
                                                </span>
                                                {isActive && (
                                                    <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-green-100 text-green-700">ACTIVE</span>
                                                )}
                                            </div>
                                            <p className="text-xs text-slate-500 mt-0.5">
                                                {format(new Date(d.start_date), 'MMM d, yyyy')} — {format(new Date(d.end_date), 'MMM d, yyyy')}
                                                {d.reason && ` · ${d.reason}`}
                                            </p>
                                        </div>
                                        <button
                                            onClick={() => onDeleteDelegation(d.id)}
                                            className="p-1.5 rounded-md hover:bg-red-50 text-slate-400 hover:text-red-500"
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </button>
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>

                {/* New Delegation Form */}
                {dept.hod_id ? (
                    <div className="border-t border-slate-200 pt-4 space-y-3">
                        <h4 className="text-sm font-semibold text-slate-700">Create New Delegation</h4>
                        <p className="text-xs text-slate-500">
                            Delegate HOD responsibilities to another user during a specific period (e.g. while on leave).
                        </p>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Acting HOD</label>
                            <select
                                value={delegateId}
                                onChange={(e) => setDelegateId(e.target.value)}
                                className="w-full h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                            >
                                <option value="">Select a user...</option>
                                {eligibleDelegates.map(u => (
                                    <option key={u.id} value={u.id}>
                                        {u.first_name} {u.last_name} — {getRoleLabel(u.role as UserRole)}
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Start Date</label>
                                <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
                            </div>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">End Date</label>
                                <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
                            </div>
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-slate-700">Reason (Optional)</label>
                            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. HOD on annual leave" />
                        </div>
                        <Button onClick={handleCreate} isLoading={saving} className="w-full">
                            Create Delegation
                        </Button>
                    </div>
                ) : (
                    <div className="border-t border-slate-200 pt-4">
                        <p className="text-sm text-amber-600 bg-amber-50 p-3 rounded-lg">
                            ⚠️ Assign an HOD to this department first before creating delegations.
                        </p>
                    </div>
                )}
            </div>
        </Modal>
    );
};

export default DepartmentManagement;
