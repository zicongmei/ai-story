document.addEventListener('DOMContentLoaded', () => {
    const apiKeyInput = document.getElementById('apiKey');
    const modelSelect = document.getElementById('modelSelect');
    const systemInstructionTextarea = document.getElementById('systemInstruction');
    const nextParagraphPromptTextarea = document.getElementById('nextParagraphPrompt');
    const storyOutputTextarea = document.getElementById('storyOutput');
    const generateBtn = document.getElementById('generateBtn');
    const clearAllBtn = document.getElementById('clearAllBtn'); // New: Get clear button
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorDisplay = document.getElementById('errorDisplay');

    const GEMINI_API_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models/';

    const defaultSystemInstruction = 'You are a skilled story writer. Continue the story one paragraph at a time, keeping the tone consistent. Ensure the new paragraph naturally follows the existing text and incorporates the given prompt for the next part of the story.';

    // Load saved settings from localStorage
    apiKeyInput.value = localStorage.getItem('geminiApiKey') || '';
    modelSelect.value = localStorage.getItem('geminiModel') || 'gemini-2.5-flash-lite';
    systemInstructionTextarea.value = localStorage.getItem('geminiSystemInstruction') || defaultSystemInstruction;
    nextParagraphPromptTextarea.value = localStorage.getItem('geminiNextParagraphPrompt') || ''; // New: Load next paragraph prompt
    storyOutputTextarea.value = localStorage.getItem('geminiStoryOutput') || ''; // New: Load story output

    // Save settings to localStorage on change
    apiKeyInput.addEventListener('input', () => localStorage.setItem('geminiApiKey', apiKeyInput.value));
    modelSelect.addEventListener('change', () => localStorage.setItem('geminiModel', modelSelect.value));
    systemInstructionTextarea.addEventListener('input', () => localStorage.setItem('geminiSystemInstruction', systemInstructionTextarea.value));
    nextParagraphPromptTextarea.addEventListener('input', () => localStorage.setItem('geminiNextParagraphPrompt', nextParagraphPromptTextarea.value)); // New: Save next paragraph prompt
    storyOutputTextarea.addEventListener('input', () => localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value)); // New: Save story output


    generateBtn.addEventListener('click', generateParagraph);
    clearAllBtn.addEventListener('click', clearAllContents); // New: Add event listener for clear button

    function clearAllContents() {
        if (!confirm('Are you sure you want to clear all contents and settings? This cannot be undone.')) {
            return;
        }

        // Clear input fields
        apiKeyInput.value = '';
        modelSelect.value = 'gemini-2.5-flash-lite'; // Reset to default model
        systemInstructionTextarea.value = defaultSystemInstruction; // Reset to default instruction
        nextParagraphPromptTextarea.value = '';
        storyOutputTextarea.value = '';

        // Clear localStorage
        localStorage.removeItem('geminiApiKey');
        localStorage.removeItem('geminiModel');
        localStorage.removeItem('geminiSystemInstruction');
        localStorage.removeItem('geminiNextParagraphPrompt');
        localStorage.removeItem('geminiStoryOutput');

        showError(''); // Clear any displayed errors
    }

    async function generateParagraph() {
        const apiKey = apiKeyInput.value.trim();
        const selectedModel = modelSelect.value;
        const systemInstruction = systemInstructionTextarea.value.trim();
        const currentStory = storyOutputTextarea.value.trim();
        const nextParagraphPrompt = nextParagraphPromptTextarea.value.trim();

        if (!apiKey) {
            showError('Please enter your Gemini API Key.');
            return;
        }

        generateBtn.disabled = true;
        loadingIndicator.classList.remove('hidden');
        showError(''); // Clear previous errors

        let userPrompt = '';
        if (currentStory === '') {
            userPrompt = `Start a new story. The first paragraph should be about: ${nextParagraphPrompt}`;
        } else {
            userPrompt = `Here is the story so far:\n\n${currentStory}\n\nWhat should happen next is: ${nextParagraphPrompt}\n\nContinue the story with ONE new paragraph, making sure it logically follows the previous text and incorporates the "what should happen next" prompt.`;
        }
        
        const requestBody = {
            contents: [{
                role: 'user',
                parts: [{ text: userPrompt }]
            }],
            generationConfig: {
                temperature: 0.9,
                topP: 1,
                topK: 1,
                maxOutputTokens: 500, // Adjust as needed for paragraph length
            },
        };

        if (systemInstruction) {
            requestBody.systemInstruction = {
                parts: [{ text: systemInstruction }]
            };
        }

        try {
            const response = await fetch(`${GEMINI_API_BASE_URL}${selectedModel}:generateContent?key=${apiKey}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error ? errorData.error.message : response.statusText);
            }

            const data = await response.json();
            const generatedText = data.candidates[0]?.content?.parts[0]?.text;

            if (generatedText) {
                if (storyOutputTextarea.value.trim() === '') {
                    storyOutputTextarea.value = generatedText.trim();
                } else {
                    storyOutputTextarea.value += '\n\n' + generatedText.trim();
                }
                // Clear the next paragraph prompt after generation
                nextParagraphPromptTextarea.value = '';
                // Scroll to the bottom of the story output
                storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
            } else {
                showError('No content generated. The model might have been blocked due to safety concerns or returned an empty response.');
            }

        } catch (error) {
            console.error('Error calling Gemini API:', error);
            showError(`Failed to generate paragraph: ${error.message}`);
        } finally {
            generateBtn.disabled = false;
            loadingIndicator.classList.add('hidden');
        }
    }

    function showError(message) {
        if (message) {
            errorDisplay.textContent = message;
            errorDisplay.classList.remove('hidden');
        } else {
            errorDisplay.textContent = '';
            errorDisplay.classList.add('hidden');
        }
    }
});