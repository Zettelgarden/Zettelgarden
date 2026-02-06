import React, { useState, useEffect } from "react";
import { Button } from "../components/Button";
import ReactMarkdown from "react-markdown";
import { createSummarization, fetchSummarization, SummarizeJobResponse } from "../api/summarizer";

export function Summarizer() {
    const [text, setText] = useState("");
    const [jobId, setJobId] = useState<number | null>(null);
    const [status, setStatus] = useState<string | null>(null);
    const [result, setResult] = useState<string | null>(null);
    const [inputJobId, setInputJobId] = useState("");

    async function startSummarization() {
        setJobId(null);
        setStatus(null);
        setResult(null);

        try {
            const data = await createSummarization(text);
            setJobId(data.id);
            setStatus(data.status);
        } catch (err) {
            console.error(err);
            setStatus("error");
        }
    }

    useEffect(() => {
        let interval: ReturnType<typeof setInterval>;
        if (jobId && status !== "complete" && status !== "failed") {
            interval = setInterval(async () => {
                try {
                    const data = await fetchSummarization(jobId);
                    setStatus(data.status);
                    if (data.status === "complete" || data.status === "failed") {
                        setResult(data.result || "");
                        clearInterval(interval);
                    }
                } catch (err) {
                    console.error(err);
                }
            }, 2000);
        }
        return () => {
            if (interval) clearInterval(interval);
        };
    }, [jobId, status]);

    async function loadSummarizationById() {
        try {
            const id = parseInt(inputJobId, 10);
            if (isNaN(id)) throw new Error("Invalid ID");
            const data = await fetchSummarization(id);
            setJobId(id);
            setStatus(data.status);
            setResult(data.result ?? null);
        } catch (err) {
            console.error(err);
            setStatus("error");
            setResult(null);
        }
    }

    return (
        <div className="p-4">
            <h2 className="text-xl font-semibold mb-4">Summarize Text</h2>
            <textarea
                className="w-full min-h-[150px] max-h-[40vh] sm:max-h-none border border-slate-400 rounded p-2 mb-2 resize-y"
                rows={6}
                placeholder="Enter text to summarize..."
                value={text}
                onChange={(e) => setText(e.target.value)}
            />
            <div className="mb-4">
                <Button onClick={startSummarization}>Submit for Summarization</Button>
            </div>

            <div className="mt-6">
                <h3 className="text-lg font-semibold mb-2">Load by Job ID</h3>
                <input
                    className="border border-slate-400 rounded p-2 mr-2"
                    type="text"
                    placeholder="Enter job ID"
                    value={inputJobId}
                    onChange={(e) => setInputJobId(e.target.value)}
                />
                <Button onClick={loadSummarizationById}>Load</Button>
            </div>

            {jobId && (
                <div className="p-2 border rounded bg-slate-100 mb-4">
                    <p><strong>Job ID:</strong> {jobId}</p>
                    <p><strong>Status:</strong> {status}</p>
                </div>
            )}

            {result && (
                <div className="prose max-w-full border rounded bg-white p-4">
                    <ReactMarkdown>{result}</ReactMarkdown>
                </div>
            )}
        </div>
    );
}
