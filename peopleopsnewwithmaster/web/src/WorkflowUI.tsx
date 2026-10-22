import { useEffect, useId, useState } from 'react';
async function api(path: string, body?: unknown) {
    const r = await fetch('/api/v1' + path, { credentials: 'include', headers: { 'Content-Type': 'application/json' }, ...(body ? { method: 'POST', body: JSON.stringify(body) } : {}) });
    const result = await r.json();
    if (!r.ok)
        throw Error(result.error?.message || 'Request failed');
    return result;
}
export function EmployeeSearch({ value, onChange, onSelect }: {
    value: string;
    onChange: (id: string) => void;
    onSelect?: (employee: any) => void;
}) {
    const id = useId();
    const [query, setQuery] = useState('');
    const [items, setItems] = useState<any[]>([]);
    const [error, setError] = useState('');
    const [busy, setBusy] = useState(false);
    const [active, setActive] = useState(0);
    useEffect(() => { if (!value && onSelect)
        setQuery(''); }, [value]);
    useEffect(() => {
        let cancelled = false;
        if (value || !query.trim()) {
            setItems([]);
            setBusy(false);
            return;
        }
        const timer = setTimeout(() => { setBusy(true); api('/employees?q=' + encodeURIComponent(query) + '&limit=20').then(r => { if (!cancelled) {
            setItems(r.items);
            setActive(0);
            setError('');
        } }).catch(e => { if (!cancelled)
            setError(e.message); }).finally(() => { if (!cancelled)
            setBusy(false); }); }, 250);
        return () => { cancelled = true; clearTimeout(timer); };
    }, [query, value]);
    const choose = (e: any) => { setQuery(onSelect ? "" : `${e.firstName} ${e.lastName} · ${e.employeeNumber}`); setItems([]); onChange(e.id); onSelect?.(e); };
    return <div className="employee-combobox"><div className="search-input-row"><input role="combobox" aria-label="Find employee" aria-autocomplete="list" aria-expanded={items.length > 0} aria-controls={id} aria-activedescendant={items.length ? `${id}-${active}` : undefined} value={query} placeholder="Search name, employee number or email" autoComplete="off" onChange={e => { onChange(''); setQuery(e.target.value); }} onKeyDown={e => { if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActive(i => Math.min(items.length - 1, i + 1));
    } if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActive(i => Math.max(0, i - 1));
    } if (e.key === 'Enter' && items[active]) {
        e.preventDefault();
        choose(items[active]);
    } if (e.key === 'Escape')
        setItems([]); }}/>{(query || value) && <button type="button" className="secondary" onClick={() => { onChange(''); setQuery(''); setItems([]); }}>Clear</button>}</div>
 {items.length > 0 && <div id={id} role="listbox" className="employee-search-results">{items.map((e, i) => <button type="button" id={`${id}-${i}`} key={e.id} role="option" aria-selected={i === active} className={i === active ? 'selected-row' : ''} onClick={() => choose(e)}>{e.firstName} {e.lastName}<small>{e.employeeNumber} · {e.department}</small></button>)}</div>}
 {busy && <small role="status">Searching…</small>}{error && <small role="alert">{error}</small>}{query && !value && !busy && !items.length && !error && <small>Select a matching employee from the search results.</small>}
 </div>;
}
export function BulkShiftPanel({ template, onClose, onSaved }: {
    template: any;
    onClose: () => void;
    onSaved: () => void;
}) {
    const [selected, setSelected] = useState<any[]>([]);
    const [date, setDate] = useState(''), [endDate, setEndDate] = useState(''), [note, setNote] = useState('');
    const [busy, setBusy] = useState(false), [results, setResults] = useState<any[]>([]), [error, setError] = useState('');
    const [previewKey, setPreviewKey] = useState(''), [department, setDepartment] = useState(''), [matches, setMatches] = useState<any[]>([]);
    const selectionKey = JSON.stringify([template.id, selected.map(e => e.id).sort(), date, endDate, note]);
    useEffect(() => { setResults([]); setPreviewKey(''); }, [selectionKey]);
    useEffect(() => { let cancelled = false; if (!department.trim()) {
        setMatches([]);
        return;
    } const timer = setTimeout(() => { api('/employees?q=' + encodeURIComponent(department) + '&limit=200').then(r => { if (!cancelled)
        setMatches(r.items.filter((e: any) => e.status === 'Active' && e.department.toLowerCase().includes(department.toLowerCase()))); }).catch(e => { if (!cancelled)
        setError(e.message); }); }, 250); return () => { cancelled = true; clearTimeout(timer); }; }, [department]);
    async function submit(preview: boolean) { setBusy(true); setError(''); try {
        const r = await api('/shift-assignments/bulk', { shiftId: template.id, employeeIds: selected.map(e => e.id), date, endDate: endDate || date, note, preview });
        setResults(r.results);
        setPreviewKey(preview ? selectionKey : '');
        if (!preview)
            onSaved();
    }
    catch (e) {
        setError(e instanceof Error ? e.message : 'Assignment failed');
    }
    finally {
        setBusy(false);
    } }
    const label = (id: string) => { const e = selected.find(x => x.id === id); return e ? `${e.firstName} ${e.lastName}` : id; };
    return <section className="panel bulk-assignment" aria-label="Assign selected shift"><div className="panel-heading"><div><h3>Assign {template.name}</h3><p>{template.shiftType === 'OT' ? 'Overtime' : 'Regular'} · {template.startTime} – {template.endTime}</p></div><button className="secondary" disabled={busy} onClick={onClose}>Close</button></div>
 <form className="modal-form" onSubmit={e => { e.preventDefault(); submit(previewKey !== selectionKey); }}>
 <fieldset disabled={busy} className="assignment-fields">
 <label>Add employees<EmployeeSearch value="" onChange={() => { }} onSelect={e => { setSelected(old => old.some(x => x.id === e.id) ? old : [...old, e]); }}/></label>
 <label>Find a department<input value={department} onChange={e => setDepartment(e.target.value)} placeholder="Type department name"/></label>
 {matches.length > 0 && <div className="department-matches"><button type="button" className="secondary" onClick={() => setSelected(old => [...old, ...matches.filter(e => !old.some(x => x.id === e.id))])}>Select {matches.length} matching employees</button>{matches.map(e => <label className="check-label" key={e.id}><input type="checkbox" checked={selected.some(x => x.id === e.id)} onChange={ev => setSelected(old => ev.target.checked ? [...old, e] : old.filter(x => x.id !== e.id))}/>{e.firstName} {e.lastName} · {e.employeeNumber}</label>)}{matches.length === 200 && <small>Showing the first 200 matches. Narrow the department search for more specific results.</small>}</div>}
 <div className="selected-employees">{selected.map(e => <button type="button" className="secondary" key={e.id} onClick={() => setSelected(old => old.filter(x => x.id !== e.id))}>{e.firstName} {e.lastName} · {e.employeeNumber} ×</button>)}</div>
 <div className="form-grid"><label>Start date<input type="date" required value={date} onChange={e => setDate(e.target.value)}/></label><label>End date (optional)<input type="date" min={date} value={endDate} onChange={e => setEndDate(e.target.value)}/></label></div>
 <label>Note<input value={note} maxLength={1000} onChange={e => setNote(e.target.value)}/></label>
 </fieldset>
 <p className="form-hint">Assign every date in the selected range, up to 31 days and 500 assignments. Preview checks existing shifts, including overnight overlaps. Conflicts are skipped and never replaced.</p>
 <div className="modal-actions"><button className="secondary" type="button" disabled={busy || !selected.length || !date || !template.active} onClick={() => submit(true)}>Preview conflicts</button><button className="primary" disabled={busy || !selected.length || !date || !template.active || previewKey !== selectionKey}>{busy ? 'Processing…' : `Assign to ${selected.length} employees`}</button></div>
 {error && <p className="error-banner" role="alert">{error}</p>}
 {results.length > 0 && <div className="batch-results" role="status">{results.map(r => <p key={r.employeeId + ':' + r.date}><b>{label(r.employeeId)}</b> · {r.date}: {r.message}</p>)}</div>}
 </form></section>;
}
